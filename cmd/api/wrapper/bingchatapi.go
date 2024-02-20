package wrapper

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
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
	options, newMessages, err := ParseOptions(request.Messages)
	request.Messages = newMessages
	if err != nil {
		return
	}

	// Clean messages from generated content (source-attributions, search-info, ..)
	for _, message := range request.Messages {
		newMessage := filterMessage(message)
		message.Content = newMessage.Content
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
		responseNormalHistory := make([]types.BingChatResponseNormal, 0, 20)

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

					responseNormalHistory = append(responseNormalHistory, bingChatResponse)

					choiceIndex := uint(0)
					for _, argument := range bingChatResponse.Arguments {
						for _, message := range argument.Messages {
							textDelta := ""

							switch message.MessageType {
							case messagetype.Message:
								// Check preconditions
								if len(responseNormalHistory) > 1 {
									previousMessage := responseNormalHistory[len(responseNormalHistory)-2]
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
									message.Text = strings.TrimRight(message.Text, "\t\r\n")

									if strings.HasPrefix(message.Text, textMsgBuffer) {
										// When msg text starts with previous text, use diff
										textDelta += message.Text[len(textMsgBuffer):]
									} else {
										// Otherwise its a new text
										// TODO: This shouldnt really happen and will mess things up if happens more than once! Might aswell go for efficiency and only store lastMsgTextLength?
										textDelta += message.Text
									}

									var forceWait bool
									newTextDelta, forceWait := DeepLeoFormatting(textDelta)
									textDelta = newTextDelta
									if forceWait {
										continue
									}

									textMsgBuffer = message.Text

								default:
									textDelta += fmt.Sprintf("\n\n---\n[ERROR] %s:\n%s\n", message.ContentOrigin, message.HiddenText)
								}

							case messagetype.InternalSearchQuery:
								textDelta = fmt.Sprintf("- Search: `%s`\n\n", message.HiddenText)
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

					var chunk models.CompletionChunk
					switch bingChatResponse.Item.Result.Value {
					case types.Success:
						// Construct source-Attributes for markdown
						var delta *models.Message
						if len(bingChatResponse.Item.Messages) > 0 {
							lastMessage := bingChatResponse.Item.Messages[len(bingChatResponse.Item.Messages)-1]

							textDelta := "\n"
							for i, sourceAttribution := range lastMessage.SourceAttributions {
								textDelta += fmt.Sprintf("\n[src%d]: %s", i+1, sourceAttribution.SeeMoreUrl)
							}

							delta = &models.Message{
								Role:    lastMessage.Author,
								Content: textDelta,
							}
						}

						chunk = models.CompletionChunk{
							ID:     bingChatResponse.Item.RequestId,
							Object: "chat.completion.chunk",
							Model:  workItem.Model,
							Choices: []models.CompletionChunkChoice{
								{
									FinishReason: "stop",
									Delta:        *delta,
								},
							},
						}

					default:
						chunk = models.CompletionChunk{
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

					outputCh <- chunk

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

func ParseOptions(messages []models.Message) (options types.Options, newMessages []models.Message, err error) {
	for _, message := range messages {
		if strings.EqualFold(message.Role, "system") {
			match := optionSearchRegexp.FindStringSubmatch(message.Content)
			if len(match) >= 3 {
				options.SearchEnabled, err = strconv.ParseBool(match[2])
				if err != nil {
					return
				}

				message.Content = optionSearchRegexp.ReplaceAllString(message.Content, "")
				message.Content = strings.Trim(message.Content, " \r\n\t")
			}
		}
		if message.Content != "" {
			newMessages = append(newMessages, message)
		}
	}
	return
}

func filterMessage(message models.Message) models.Message {
	// Filter some aspects to reduce halucinations and token-length
	switch message.Role {
	case "assistant", "bot":
		message.Content = regexp.MustCompile(`(?s)\n\n\[src\d+\]: .+$`).ReplaceAllString(message.Content, "")
		//message.Content = regexp.MustCompile(` ?\[\(\d+\)\]\[src\d+\]`).ReplaceAllString(message.Content, "")
		message.Content = regexp.MustCompile(` ?\[(.+?)\]\[src\d+\]:.+(\n|$)`).ReplaceAllString(message.Content, "") // Might not be wanted?
		message.Content = regexp.MustCompile(` ?\[(.+?)\]\[src\d+\]`).ReplaceAllString(message.Content, "")
		message.Content = regexp.MustCompile(`^(- Search: .+\n\n)+`).ReplaceAllString(message.Content, "")
		message.Content = regexp.MustCompile(`^\^\[\]\(BCN\)\n`).ReplaceAllString(message.Content, "")
	}

	return message
}

// Replace links to more-supported anchors
/*
	if regexp.MustCompile(`(\[\^\d+\^|\[\^\d+|\[\^|\[)$`).MatchString(text) ||
		regexp.MustCompile(`(\]\(\^(\d+)\^|\]\(\^(\d+)|\]\(\^|\]\(|\])$`).MatchString(text) {
		continue
	}
	text = regexp.MustCompile(` ?\[\^(\d+)\^\]`).ReplaceAllString(text, " [($1)][src$1]")
	text = regexp.MustCompile(`\]\(\^(\d+)\^\)`).ReplaceAllString(text, "][src$1]")
*/
type FormattingRule struct {
	name           string
	forceWaitMatch *regexp.Regexp
	replace        FormattingReplaceRule
}
type FormattingReplaceRule struct {
	match   *regexp.Regexp
	replace string
}

var (
	formattingRules = []FormattingRule{
		{
			name:           "Link-Ref",
			forceWaitMatch: regexp.MustCompile(`(\[\^\d+\^|\[\^\d+|\[\^|\[)$`),
			replace: FormattingReplaceRule{
				match:   regexp.MustCompile(` ?\[\^(\d+)\^\]`),
				replace: " [($1)][src$1]",
			},
		},
		{
			name:           "Word-Link-Ref",
			forceWaitMatch: regexp.MustCompile(`(\]\(\^(\d+)\^|\]\(\^(\d+)|\]\(\^|\]\(|\])$`),
			replace: FormattingReplaceRule{
				match:   regexp.MustCompile(`\]\(\^(\d+)\^\)`),
				replace: "][src$1]",
			},
		},
	}
)

// Format deepleo messages and optionally also forcing a wait so things can be replaced cleanly
func DeepLeoFormatting(msg string) (newMsg string, forceWait bool) {
	for _, rule := range formattingRules {
		if rule.forceWaitMatch.MatchString(msg) {
			forceWait = true
			continue
		}

		msg = rule.replace.match.ReplaceAllString(msg, rule.replace.replace)
	}

	return msg, forceWait
}
