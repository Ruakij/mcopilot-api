package wrapper

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"git.ruekov.eu/ruakij/mcopilot-api/cmd/api/logger"
	"git.ruekov.eu/ruakij/mcopilot-api/cmd/api/models"
	"git.ruekov.eu/ruakij/mcopilot-api/lib/bingchatapi/types"
	"git.ruekov.eu/ruakij/mcopilot-api/lib/bingchatapi/types/contentorigin"
	"git.ruekov.eu/ruakij/mcopilot-api/lib/bingchatapi/types/messagetype"
)

type BingChatWrapper struct {
	api types.Api
}

func NewBingChatApiWrapper(api types.Api) *BingChatWrapper {
	return &BingChatWrapper{
		api: api,
	}
}

func (wrapper *BingChatWrapper) Init() {
	wrapper.api.Init()
}

func (wrapper *BingChatWrapper) ProcessRequest(controllerContext context.Context, request models.ChatRequest, outputCh chan<- models.CompletionChunk) (err error) {
	logger.Info.Println("ProcessRequest")
	apiOutputCh := make(chan []byte)

	// Options
	options, err := ParseOptions(request.Messages)
	if err != nil {
		return
	}

	wrapperContext, contextCancel := context.WithCancel(controllerContext)

	// Build WorkItem
	typesMessages := make([]types.Message, len(request.Messages))
	for i, message := range request.Messages {
		typesMessages[i] = types.Message(message)
	}

	workItem := types.WorkItem{
		Context:      wrapperContext,
		OutputStream: apiOutputCh,
		Messages:     typesMessages,
		Model:        request.Model,
		Options:      options,
	}

	// Start api-processing
	wrapper.api.ProcessRequest(&workItem)

	// Start translation
	go func() {
		defer close(outputCh)
		defer contextCancel()

		textMsgBuffer := ""
		responseHistory := make([]types.BingChatResponseNormal, 10)

		for {
			select {
			case <-controllerContext.Done():
				return

			case record, ok := <-apiOutputCh:
				if !ok {
					return
				}

				var bingChatResponseBasic types.BingChatResponseBasic
				err := json.Unmarshal([]byte(record), &bingChatResponseBasic)
				if err != nil {
					logger.Info.Println(err)
					continue
				}

				switch bingChatResponseBasic.Type {
				case 1:
					// Normal
					var bingChatResponse types.BingChatResponseNormal
					err := json.Unmarshal([]byte(record), &bingChatResponse)
					if err != nil {
						logger.Info.Println(err)
						continue
					}

					responseHistory = append(responseHistory, bingChatResponse)

					choiceIndex := uint(0)
					for _, argument := range bingChatResponse.Arguments {
						for _, message := range argument.Messages {
							textDelta := ""

							switch message.MessageType {
							case messagetype.Message:
								// Check preconditions
								if len(responseHistory) > 1 {
									previousMessage := responseHistory[len(responseHistory)-2]
									if len(previousMessage.Arguments) > 0 {
										lastArgumentFromPreviousMessage := previousMessage.Arguments[len(previousMessage.Arguments)-1]
										if len(lastArgumentFromPreviousMessage.Messages) > 0 {
											lastMessageFromlastArgumentFromPreviousMessage := lastArgumentFromPreviousMessage.Messages[len(lastArgumentFromPreviousMessage.Messages)-1]
											switch lastMessageFromlastArgumentFromPreviousMessage.MessageType {
											case messagetype.InternalSearchResult:
												textDelta += "\n"
											}
										}
									}
								}

								switch message.ContentOrigin {
								case contentorigin.DeepLeo:
									// Remove trailing newline from text. If its last, it wont be there, if its not, will be send again.
									message.Text = strings.TrimRight(message.Text, "\t\r\n ")

									if strings.HasPrefix(message.Text, textMsgBuffer) {
										// When msg text starts with previous text, use diff
										textDelta += message.Text[len(textMsgBuffer):]
									} else {
										// Otherwise its a new text
										// TODO: This shouldnt really happen and will mess things up if happens more than once! Might aswell go for efficiency and only store lastMsgTextLength?
										textDelta += message.Text
									}
									textMsgBuffer = message.Text

								default:
									textDelta += fmt.Sprintf("\n\n---\n[ERROR] %s:\n%s\n", message.ContentOrigin, message.HiddenText)
								}

							case messagetype.InternalSearchQuery:
								textDelta = fmt.Sprintf("- Search: `%s`\n", message.HiddenText)

							}

							if textDelta != "" {
								outputCh <- models.CompletionChunk{
									ID:     argument.RequestId,
									Object: "chat.completion.chunk",
									Model:  workItem.Model,
									Choices: []models.CompletionChunkChoice{
										{
											Index: choiceIndex,
											Delta: models.Message{
												Role:    "bot",
												Content: textDelta,
											},
										},
									},
								}
								choiceIndex++
							}
						}
					}

				case 2:
					// Summary
					var bingChatResponse types.BingChatResponseSummary
					err := json.Unmarshal([]byte(record), &bingChatResponse)
					if err != nil {
						logger.Info.Println(err)
						continue
					}

					switch bingChatResponse.Item.Result.Value {
					case types.Success:
						outputCh <- models.CompletionChunk{
							ID:     bingChatResponse.Item.RequestId,
							Object: "chat.completion.chunk",
							Model:  workItem.Model,
							Choices: []models.CompletionChunkChoice{
								{
									FinishReason: "stop",
								},
							},
						}

					default:
						outputCh <- models.CompletionChunk{
							ID:     bingChatResponse.Item.RequestId,
							Object: "chat.completion.chunk",
							Model:  workItem.Model,
							Choices: []models.CompletionChunkChoice{
								{
									Delta: models.Message{
										Role:    "bot",
										Content: fmt.Sprintf("\n\n---\n[ERROR] %s:\n%s\n", string(bingChatResponse.Item.Result.Value), bingChatResponse.Item.Result.Message),
									},
									FinishReason: string(bingChatResponse.Item.Result.Value),
								},
							},
						}
					}

				case 3:
					// End
					return
				}

			}
		}
	}()

	return
}

var (
	optionSearchRegexp *regexp.Regexp = regexp.MustCompile(`(?i)(^|\n)/search (.+)$`)
)

func ParseOptions(messages []models.Message) (options types.Options, err error) {
	for _, message := range messages {
		if strings.EqualFold(message.Role, "system") {
			match := optionSearchRegexp.FindStringSubmatch(message.Content)
			if len(match) >= 3 {
				options.SearchEnabled, err = strconv.ParseBool(match[2])
				if err != nil {
					return
				}
			}
		}
	}
	return
}
