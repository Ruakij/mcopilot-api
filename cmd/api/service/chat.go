package ChatService

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"git.ruekov.eu/ruakij/mcopilot-api/cmd/api/models"
	browsercontroller "git.ruekov.eu/ruakij/mcopilot-api/cmd/browserController"
)

func ProcessChatRequest(context context.Context, request models.ChatRequest, events chan<- models.CompletionChunk, resultChan chan<- models.Completion) {
	returnChan := make(chan browsercontroller.BingChatResponse, 200)

	var completion models.Completion
	go func() {
		var completionChunk models.CompletionChunk
		fullText := ""
		previousText := ""
		lastSourceAttributions := []browsercontroller.SourceAttributions{}
		for {
			select {
			case <-context.Done():

				return
			case bingChatResponse, ok := <-returnChan:
				if ok {
					lastArgument := bingChatResponse.Arguments[len(bingChatResponse.Arguments)-1]

					textDelta := ""

					switch bingChatResponse.Type {
					case 1:
						text := ""
						if len(lastArgument.Messages) > 0 {
							message := lastArgument.Messages[len(lastArgument.Messages)-1]
							//for _, message := range lastArgument.Messages {
							// Skip message types we dont want to return
							switch message.MessageType {
							case browsercontroller.BingChatMessageType(browsercontroller.InternalSearchResult),
								browsercontroller.BingChatMessageType(browsercontroller.AdsQuery),
								browsercontroller.BingChatMessageType(browsercontroller.InternalLoaderMessage):
								continue
							}

							if text != "" {
								text += "\n"
							}

							// Add message
							switch message.MessageType {
							case browsercontroller.InternalSearchQuery:
								text += fmt.Sprintf("- Search: %s\n\n", message.HiddenText)

							default:
								text += message.Text

								// Add hidden Message
								if message.HiddenText != "" && message.Text != message.HiddenText {
									if message.Text != "" {
										text += "\n"
									}
									text += fmt.Sprintf("\n*%s*\n", message.HiddenText)
								}
							}

							if len(message.SourceAttributions) > 0 {
								lastSourceAttributions = message.SourceAttributions
							}

							//}

							fmt.Println(text)

							// Replace links to more-supported anchors
							if regexp.MustCompile(`(\[\^\d+\^|\[\^\d+|\[\^|\[)$`).MatchString(text) ||
								regexp.MustCompile(`(\]\(\^(\d+)\^|\]\(\^(\d+)|\]\(\^|\]\(|\])$`).MatchString(text) {
								continue
							}
							text = regexp.MustCompile(` ?\[\^(\d+)\^\]`).ReplaceAllString(text, " [($1)][src$1]")
							text = regexp.MustCompile(`\]\(\^(\d+)\^\)`).ReplaceAllString(text, "][src$1]")

							textDelta = text
							if message.MessageType == browsercontroller.Message {
								// Delta based on previous as type "message" comes as full string instead of deltas
								textDelta = ""
								if len(previousText) < len(text) {
									textDelta = text[len(previousText):]
								}
								previousText = text
							}
						}

					case 2:
						message := lastArgument.Messages[len(lastArgument.Messages)-1]
						switch message.ContentOrigin {
						case browsercontroller.JailBreakClassifier:
							textDelta += "\n" + message.Text

							if message.HiddenText != "" && message.Text != message.HiddenText {
								if message.Text != "" {
									textDelta += "\n"
								}
								textDelta += fmt.Sprintf("\n*%s*", message.HiddenText)
							}
						}
					}

					// Build CompletionChunk
					completionChunk = models.CompletionChunk{
						ID:                lastArgument.RequestId,
						Object:            "chat.completion.chunk",
						Created:           0,
						Model:             "BingChat-Creative",
						SystemFingerprint: "",
						Choices: []models.CompletionChunkChoice{
							{
								Index: 0,
								Delta: models.Message{
									Content: textDelta,
								},
							},
						},
					}
					if fullText == "" {
						completionChunk.Choices[len(completionChunk.Choices)-1].Delta.Role = "assistant"
					}

					fullText = fullText + textDelta

					if request.Stream {
						events <- completionChunk
					}
				} else {
					// Send last stream
					sourceText := ""
					if len(lastSourceAttributions) > 0 {
						sourceText += "\n\n"
						for i, sourceAttribution := range lastSourceAttributions {
							sourceText += fmt.Sprintf("[src%d]: %s\n", i+1, sourceAttribution.SeeMoreUrl)
						}
					}
					completionChunk.Choices = []models.CompletionChunkChoice{
						{
							Delta: models.Message{
								Content: sourceText,
							},
							Index:        0,
							FinishReason: "stop",
						},
					}

					if request.Stream {
						events <- completionChunk
					}
					close(events)

					// Build completion
					completion = models.Completion{
						ID:      completionChunk.ID,
						Object:  "chat.completion",
						Created: 0,
						Model:   completionChunk.Model,
						Choices: []models.Choice{
							{
								Index: 0,
								Message: models.Message{
									Role:    "bot",
									Content: fullText,
								},
								FinishReason: "stop",
							},
						},
					}
					resultChan <- completion

					return
				}
			}
		}
	}()

	// Build message
	messages := make([]string, 0, len(request.Messages))
	for _, message := range request.Messages {
		msgContent := message.Content

		// Filter some aspects to reduce halucinations and token-length
		switch message.Role{
		case "assistant":
			msgContent = regexp.MustCompile(`(?s)\n\n\[src\d+\]: https?:\/\/.+$`).ReplaceAllString(msgContent, "")
			msgContent = regexp.MustCompile(` ?\[\(\d+\)\]\[src\d+\]`).ReplaceAllString(msgContent, "")
			msgContent = regexp.MustCompile(`\[(.+)\]\[src\d+\]`).ReplaceAllString(msgContent, "$1")
			msgContent = regexp.MustCompile(`(?s)^- Search: .+\n\n`).ReplaceAllString(msgContent, "")
		}

		messages = append(messages, strings.Trim(msgContent, "\r\n\t "))
	}
	fullMessage := strings.Join(messages, "\n---\n")

	browsercontroller.ProcessChatRequest(context, fullMessage, returnChan)
}
