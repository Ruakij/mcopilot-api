package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"git.ruekov.eu/ruakij/mcopilot-api/cmd/api/models"
	browsercontroller "git.ruekov.eu/ruakij/mcopilot-api/cmd/browserController"
	"git.ruekov.eu/ruakij/mcopilot-api/lib/advancedmap"
	"github.com/go-rod/rod"
)

var sessions = advancedmap.NewAdvancedMap[string, *rod.Page](time.Minute*30, 10)

func init() {
	sessions.SetRemoveHook(func(key string, item advancedmap.Item[*rod.Page]) {
		item.Data.MustClose()
		item.Data.Close()
	})
}

func filterMessage(message models.Message) models.Message {
	// Filter some aspects to reduce halucinations and token-length
	switch message.Role {
	case "assistant":
		message.Content = regexp.MustCompile(`(?s)\n\n\[src\d+\]: .+$`).ReplaceAllString(message.Content, "")
		//message.Content = regexp.MustCompile(` ?\[\(\d+\)\]\[src\d+\]`).ReplaceAllString(message.Content, "")
		message.Content = regexp.MustCompile(` ?\[(.+?)\]\[src\d+\]:.+(\n|$)`).ReplaceAllString(message.Content, "") // Might not be wanted?
		message.Content = regexp.MustCompile(` ?\[(.+?)\]\[src\d+\]`).ReplaceAllString(message.Content, "")
		message.Content = regexp.MustCompile(`^(- Search: .+\n\n)+`).ReplaceAllString(message.Content, "")
		message.Content = regexp.MustCompile(`^\^\[\]\(BCN\)\n`).ReplaceAllString(message.Content, "")

		// Generate replies cannot be used, simply remove them to make it work
		if regexp.MustCompile(`^(- Generate: .+\n\n)+`).MatchString(message.Content) {
			message.Content = ""
		}
	}

	message.Content = strings.Trim(message.Content, "\n ")

	return message
}

func extractFilteredMessageContents(messages []models.Message) []string {
	messageContents := make([]string, 0, len(messages))

	for _, message := range messages {
		message = filterMessage(message)
		if message.Content == "" {
			continue
		}
		messageContents = append(messageContents, message.Content)
	}

	return messageContents
}

// ProcessChatRequestWithSession processes the chat request with a reuseable session
func ProcessChatRequest(context context.Context, request models.ChatRequest, events chan<- models.CompletionChunk, resultChan chan<- models.Completion) error {
	// Build message
	messageContents := extractFilteredMessageContents(request.Messages)
	messageContext := messageContents[:len(messageContents)-1]
	messageContextKey := strings.Join(messageContext, "|")

	returnChan := make(chan browsercontroller.BingChatResponse, 200)

	// Get the chat session page for the request context

	page, ok := sessions.Get(messageContextKey)
	if ok {
		// Test if still ok
		_, err := page.Eval("() => console.log('ping from master')")
		if err != nil {
			sessions.Remove(messageContextKey)
			page = nil
		}
	}

	// Remember user-request in context after we checked it
	messageContext = messageContents
	var work *browsercontroller.WorkItem
	if page != nil {
		work = &browsercontroller.WorkItem{
			Page:         page,
			Context:      context,
			Input:        request.Messages[len(request.Messages)-1].Content,
			OutputStream: returnChan,
		}
	} else {
		fullMessage := strings.Join(messageContents, "\n---\n")
		work = &browsercontroller.WorkItem{
			Page:         nil,
			Context:      context,
			Input:        fullMessage,
			OutputStream: returnChan,
		}
	}
	browsercontroller.ProcessChatRequest(work)

	// The rest of the code is the same as ProcessChatRequest
	var completion models.Completion
	go func() {
		var completionChunk models.CompletionChunk
		fullText := ""
		previousText := ""
		finishReason := ""
		lastSourceAttributions := []browsercontroller.SourceAttributions{}
		for {
			select {
			case <-context.Done():
				sessions.RemoveWithoutHooks(messageContextKey)
				// Build new context
				/*newMessage := filterMessage(models.Message{
					Role:    "assistant",
					Content: fullText,
				})
				messageContext = append(messageContext, newMessage.Content)
				messageContextKey := strings.Join(messageContext, "|")
				sessions.Put(messageContextKey, page)*/

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

							case browsercontroller.GenerateContentQuery:
								text += fmt.Sprintf("- Generate: %s\n\n", message.Text)

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
						case browsercontroller.JailBreakClassifier, browsercontroller.Aplology:
							finishReason = string(message.ContentOrigin)

							textDelta += "\n\n" + message.Text

							if message.HiddenText != "" && message.Text != message.HiddenText {
								if message.Text != "" {
									textDelta += "\n"
								}
								textDelta += fmt.Sprintf("\n*%s*", message.HiddenText)
							}

							textDelta += fmt.Sprintf("\n*Finish-Reason: %s*", finishReason)
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
								FinishReason: finishReason,
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
					/*if completionChunk.ID == "" {
						resultChan <- models.Completion{}
						return
					}*/

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
							FinishReason: completionChunk.Choices[len(completionChunk.Choices)-1].FinishReason,
						},
					}

					// Handle session-storage
					switch completionChunk.Choices[len(completionChunk.Choices)-1].FinishReason {
					case string(browsercontroller.Aplology), string(browsercontroller.JailBreakClassifier):
						sessions.Remove(messageContextKey)

					default:
						// Build new context
						newMessage := filterMessage(models.Message{
							Role:    "assistant",
							Content: fullText,
						})
						if newMessage.Content != "" {
							sessions.RemoveWithoutHooks(messageContextKey)
							messageContext = append(messageContext, newMessage.Content)
							messageContextKey := strings.Join(messageContext, "|")
							sessions.Put(messageContextKey, work.Page)
						}
					}

					// Return last data
					if request.Stream {
						events <- completionChunk
						close(events)
					} else {
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
					}

					return
				}
			}
		}
	}()

	return nil
}
