package reng

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"git.ruekov.eu/ruakij/mcopilot-api/cmd/api/logger"
	"git.ruekov.eu/ruakij/mcopilot-api/lib/bingchatapi/types"
	"git.ruekov.eu/ruakij/mcopilot-api/lib/bingchatapi/types/tone"
	"git.ruekov.eu/ruakij/mcopilot-api/lib/httpmisc"
	"github.com/gorilla/websocket"
	"storj.io/common/uuid"
)

type RengApi struct {
	workerCount int
	Cookies     map[string]string
	workQueue   chan *types.WorkItem
	hooks       Hooks
}

type Hooks struct {
	CreateConversation func(session *httpmisc.HttpClientSession, tone tone.Type, imageData *string) (conversation *Conversation, err error)
}

// Create a new Reverse-Engineered-API and starts workers
func NewRengApi(workerCount int, cookies map[string]string) *RengApi {
	if cookies == nil || len(cookies) == 0 {
		cookies = Defaults.cookies
	}
	rengApi := RengApi{
		workerCount: workerCount,
		Cookies:     cookies,
		workQueue:   make(chan *types.WorkItem),
		hooks: Hooks{
			CreateConversation: createConversation,
		},
	}

	return &rengApi
}

func (api *RengApi) SetHooks(hooks *Hooks) {
	if hooks.CreateConversation != nil {
		api.hooks.CreateConversation = hooks.CreateConversation
	}
}

func (api *RengApi) Init() {
	for i := 0; i < api.workerCount; i++ {
		go api.startWorker()
	}
}

// Place an item into the workQueue, blocks until item is in processing
func (api RengApi) ProcessRequest(workItem *types.WorkItem) {
	api.workQueue <- workItem
}

func (api RengApi) startWorker() {
	for {
		api.workerProcess()
		time.Sleep(time.Second * 1)
	}
}

func (api RengApi) workerProcess() {
	// Get next item from queue
	workItem := <-api.workQueue
	defer close(workItem.OutputStream)

	// Check is context done
	select {
	case <-workItem.Context.Done():
		return
	default:
	}

	// Convert model request to tone
	msgTone := tone.GetToneByString(workItem.Model)
	if msgTone == tone.Unknown {
		// Default is creative
		msgTone = tone.Creative
	}
	workItem.Model = string(msgTone)

	// Build messageContext
	messageContext := ""
	for _, message := range workItem.Messages[:len(workItem.Messages)-1] {
		switch message.Role {
		case "user":
			messageContext += fmt.Sprintf("%s\n---\n", message.Content)

		default:
			messageContext += fmt.Sprintf("%s:\n%s\n---\n", message.Role, message.Content)
		}
	}

	// TODO: messageContext doesnt work, so just put it to the front of messages
	if messageContext != "" {
		workItem.Messages[len(workItem.Messages)-1].Content = messageContext + workItem.Messages[len(workItem.Messages)-1].Content
	}

	messageContext = ""

	outputCh, err := api.stream_generate(workItem.Context, workItem.Messages[len(workItem.Messages)-1].Content, msgTone, nil, messageContext, "", api.Cookies, workItem.Options.SearchEnabled, true)
	if err != nil {
		logger.Error.Printf("stream_generate failed: %s", err)
		return
	}
	for {
		response, ok := <-outputCh
		if !ok {
			return
		}
		workItem.OutputStream <- response
	}
}

type Conversation struct {
	ConversationId        string
	ClientId              string
	ConversationSignature string
	imageInfo             struct {
		imageUrl         *string
		originalImageUrl *string
	}
}

type CreateConversationResponseDataResultValue string

const (
	CreateConversationResponseDataResultValueSuccess = "Success"
)

type CreateConversationResponseData struct {
	ConversationId string
	ClientId       string
	Result         struct {
		Value   CreateConversationResponseDataResultValue
		Message string
	}
}

func createConversation(session *httpmisc.HttpClientSession, tone tone.Type, imageData *string) (conversation *Conversation, err error) {
	// Create a new HTTP request
	req, err := session.NewRequest("GET", fmt.Sprintf("https://copilot.microsoft.com/turing/conversation/create?bundleVersion=%s", Defaults.bundleVersion), nil)
	if err != nil {
		return
	}

	// Set the request headers
	req.Header.Set("Accept", "application/json")
	req.Header.Set("x-ms-client-request-id", MustNewUUID().String())
	req.Header.Set("x-ms-useragent", "azsdk-js-api-client-factory/1.0.0-beta.1 core-rest-pipeline/1.12.0 OS/Linux")

	// Adjust for bingChat/Copilot?
	req.Header.Set("Referer", "https://copilot.microsoft.com/")

	// Send the request and get the response
	resp, err := session.Client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}

	// Parse response
	var responseData CreateConversationResponseData
	err = json.Unmarshal(body, &responseData)
	if err != nil {
		return
	}

	// Check response
	if responseData.Result.Value != CreateConversationResponseDataResultValueSuccess {
		err = fmt.Errorf("result-value not 'success' but '%s' with message '%s'", responseData.Result.Value, responseData.Result.Message)
		return
	}

	// Construct conversation-metadata
	conversation = &Conversation{
		ConversationId:        responseData.ConversationId,
		ClientId:              responseData.ClientId,
		ConversationSignature: resp.Header.Get("X-Sydney-Encryptedconversationsignature"),
	}

	err = conversationValid(conversation)
	if err != nil {
		err = fmt.Errorf("conversation invalid: %s", err)
		return
	}

	// TODO: Support image-input

	return
}

func conversationValid(conversation *Conversation) error {
	if len(conversation.ConversationId) == 0 {
		return errors.New("conversationId is empty")
	}
	if len(conversation.ClientId) == 0 {
		return errors.New("clientId is empty")
	}
	if len(conversation.ConversationSignature) == 0 {
		return errors.New("conversationSignature is empty")
	}
	return nil
}

// stream_generate is a function that generates a response from Bing Chat API
// based on the given prompt, tone, image, context, proxy, cookies, web_search and gpt4_turbo parameters.
func (api RengApi) stream_generate(
	context context.Context,
	prompt string,
	tone tone.Type,
	image *string,
	contextMsg string,
	proxy string,
	cookies map[string]string,
	web_search bool,
	gpt4_turbo bool,
) (<-chan []byte, error) {
	logger.Info.Println("stream_generate()")
	// Create a channel to send the response
	output := make(chan []byte)

	// Create a new HTTP client session with the given timeout and headers
	session := httpmisc.NewHttpClientSession(Defaults.headers, cookies, time.Millisecond*900)

	// Create a conversation with the given tone and image
	logger.Info.Println("Create conversation..")
	conversation, err := api.hooks.CreateConversation(session, tone, nil)
	if err != nil {
		close(output)
		return nil, err
	}
	logger.Info.Printf("Got conversation id=%s\n", conversation.ConversationId)

	// Connect to the WebSocket endpoint with the given parameters
	logger.Info.Println("Connecting to Websocket..")

	//Url, _ := url.Parse("wss://sydney.bing.com/sydney/ChatHub")
	Url, _ := url.Parse("wss://sydney.bing.com/sydney/ChatHub")
	parameters := url.Values{}
	parameters.Set("sec_access_token", conversation.ConversationSignature)
	Url.RawQuery = parameters.Encode()

	wss, err := session.WSConnect(Url.String(), nil)
	if err != nil {
		close(output)
		return nil, err
	}
	logger.Info.Println("Connected!")

	// Send the initial message with the protocol and version
	logger.Info.Println("Send initial message..")
	err = wss.WriteMessage(websocket.TextMessage, []byte("{\"protocol\":\"json\",\"version\":1}\x1e"))
	if err != nil {
		close(output)
		return nil, err
	}
	logger.Info.Println("Sent!")

	// Receive the first message from the server
	logger.Info.Println("Receiving first message..")
	_, data, err := wss.ReadMessage()
	if err != nil {
		close(output)
		return nil, err
	}
	logger.Info.Println("Received!")
	if len(data) > 2+1 {
		return nil, fmt.Errorf("invalid response on initiation: %s", string(data))
	}

	// Send the message with the prompt, tone, context, web_search and gpt4_turbo
	message := create_message(conversation, prompt, tone, contextMsg, web_search, gpt4_turbo)
	logger.Info.Printf("Sending message request.. count=%d\n", len(message))
	err = wss.WriteMessage(websocket.TextMessage, message)
	if err != nil {
		close(output)
		return nil, err
	}
	logger.Info.Println("Sent!")

	// Initialize the response text and the returned text
	/*response_txt := ""
	returned_text := ""*/

	go func() {
		defer close(output)
		defer wss.Close()

		// Start reading
		logger.Info.Println("Start reading..")
		readCh := readWssMessages(context, wss)
		for {
			select {
			case <-context.Done():
				logger.Info.Println("END: Context closed")
				// Context closed
				return

			case readResult, ok := <-readCh:
				if !ok {
					logger.Info.Println("END: Stream ended")
					// Stream ended
					return
				}
				logger.Info.Printf("Received data messageType=%d err=%s count=%d\n", readResult.messageType, readResult.err, len(readResult.data))
				if readResult.err != nil {
					continue
				}
				if readResult.messageType != websocket.TextMessage {
					continue
				}

				msg := string(readResult.data)

				// Split the message data by the delimiter
				records := strings.Split(msg, "\x1e")

				// Iterate over the objects
				for _, record := range records {
					// Skip empty objects
					if record == "" {
						continue
					}

					output <- []byte(record)

					/*
						// Parse the object as JSON
						var response map[string]interface{}
						err = json.Unmarshal([]byte(record), &response)
						if err != nil {
							return nil, err
						}

						// Check the type of the response
						switch response["type"].(float64) {
						case 1:
							// This is a message response
							// Get the messages from the arguments
							messages := response["arguments"].([]interface{})[0].(map[string]interface{})["messages"].([]interface{})

							// Get the first message
							message := messages[0].(map[string]interface{})

							// Check the content origin of the message
							if message["contentOrigin"].(string) != "Apology" {
								// This is not an apology message
								// Check if the message has adaptive cards
								if cards, ok := message["adaptiveCards"].([]interface{}); ok {
									// Get the first card
									card := cards[0].(map[string]interface{})["body"].([]interface{})[0].(map[string]interface{})

									// Check if the card has text
									if text, ok := card["text"].(string); ok {
										// Append the text to the response text
										response_txt += text
									}

									// Check if the message has a message type
									if _, ok := message["messageType"].(string); ok {
										// Get the inline text from the card
										inline_txt := card["inlines"].([]interface{})[0].(map[string]interface{})["text"].(string)

										// Append the inline text to the response text with a new line
										response_txt += inline_txt + "\n"
									}
								} else if message["contentType"] == messagetype.Image {
									// This is an image message
									// Get the query from the message text
									query := url.QueryEscape(message["text"].(string))

									// Construct the URL for the image
									url := fmt.Sprintf("\nhttps://www.bing.com/images/create?q=%s", query)

									// Append the URL to the response text
									response_txt += url

									// Set the final flag to true
									final = true
								}
							}

							// Check if the response text starts with the returned text
							if strings.HasPrefix(response_txt, returned_text) {
								// Get the new text from the response text
								new := response_txt[len(returned_text):]

								// Check if the new text is not empty
								if new != "\n" {
									// Send the new text to the output channel
									output <- new

									// Update the returned text
									returned_text = response_txt
								}
							}
						case 2:
							// This is a result response
							// Get the result from the item
							result := response["item"].(map[string]interface{})["result"].(map[string]interface{})

							// Check if the result has an error
							if _, ok := result["error"].(bool); ok {
								// Raise an exception with the result value and message
								return nil, fmt.Errorf("%s: %s", result["value"].(string), result["message"].(string))
							}

							// Return the output channel
							return output, nil
						}*/
				}
			}
		}
	}()

	// Return the output channel
	return output, nil
}

// create_message is a function that creates a message struct for Bing Chat API
// based on the given conversation, prompt, tone, context, web_search and gpt4_turbo parameters.
func create_message(conversation *Conversation, prompt string, msgTone tone.Type, context string, web_search bool, gpt4_turbo bool) []byte {
	// Generate a random request id
	request_id := MustNewUUID().String()

	request := types.GetDefaultCopilotRequest()
	request.Setup(
		msgTone,
		web_search,
		gpt4_turbo,
		prompt,
		context,
		conversation.ClientId,
		"",
		conversation.ConversationId,
		request_id,
		conversation.imageInfo.imageUrl,
		conversation.imageInfo.originalImageUrl,
	)

	// Format the struct as JSON
	data, err := json.Marshal(request)
	if err != nil {
		panic(err)
	}

	data = append(data, 0x1E)

	// Return the JSON data
	return data
}

func MustNewUUID() uuid.UUID {
	request_id_uuid, err := uuid.New()
	if err != nil {
		panic(err)
	}
	return request_id_uuid
}
