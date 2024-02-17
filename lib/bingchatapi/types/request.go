package types

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"git.ruekov.eu/ruakij/mcopilot-api/lib/bingchatapi/types/messagetype"
	"git.ruekov.eu/ruakij/mcopilot-api/lib/bingchatapi/types/tone"
)

// MessageRequest is a struct that represents a message request for Bing chat
type Request struct {
	Arguments    []Argument `json:"arguments"`
	InvocationID string     `json:"invocationId"`
	Target       string     `json:"target"`
	Type         int        `json:"type"`
}

// Argument is a struct that represents an argument for a message request
type Argument struct {
	Source                         string                    `json:"source"`
	OptionsSets                    []string                  `json:"optionsSets"`
	AllowedMessageTypes            []messagetype.MessageType `json:"allowedMessageTypes"`
	SliceIDs                       []string                  `json:"sliceIds"`
	Verbosity                      string                    `json:"verbosity"`
	Scenario                       ScenatioType              `json:"scenario"`
	Plugins                        []any                     `json:"plugins"`
	TraceID                        string                    `json:"traceId"`
	ConversationHistoryOptionsSets []string                  `json:"conversationHistoryOptionsSets"`
	IsStartOfSession               bool                      `json:"isStartOfSession"`
	RequestID                      string                    `json:"requestId"`
	MessageRequest                 MessageRequest            `json:"message"`
	Tone                           tone.Type                 `json:"tone"`
	SpokenTextMode                 string                    `json:"spokenTextMode"`
	ConversationID                 string                    `json:"conversationId"`
	Participant                    Participant               `json:"participant"`
}

type ScenatioType string

const (
	ScenarioSERP      ScenatioType = "SERP"
	ScenarioUnderside ScenatioType = "Underside"
)

type AttachedFileInfo struct {
	FileName *string `json:"fileName,omitempty"`
	FileType *string `json:"fileType,omitempty"`
}

type ContextType string

const (
	WebPage ContextType = "WebPage"
)

type PreviousMessage struct {
	Author      string                  `json:"author,omitempty"`
	Description string                  `json:"description,omitempty"`
	ContextType ContextType             `json:"contextType,omitempty"`
	MessageType messagetype.MessageType `json:"messageType,omitempty"`
	SourceName  string                  `json:"sourceName,omitempty"`
	SourceUrl   string                  `json:"sourceUrl,omitempty"`
	Locale      string                  `json:"locale,omitempty"`
	Privacy     string                  `json:"privacy,omitempty"`
}

// Message is a struct that represents a message for a message request
type MessageRequest struct {
	Locale           string            `json:"locale"`
	Market           string            `json:"market"`
	Region           string            `json:"region"`
	Location         string            `json:"location"`
	LocationHints    []LocationHints   `json:"locationHints,omitempty"`
	UserIPAddress    string            `json:"userIpAddress"`
	Timestamp        string            `json:"timestamp"`
	Author           string            `json:"author"`
	InputMethod      string            `json:"inputMethod"`
	Text             string            `json:"text"`
	MessageType      string            `json:"messageType"`
	RequestID        string            `json:"requestId"`
	MessageID        string            `json:"messageId"`
	Privacy          string            `json:"privacy"`
	OriginalImageUrl string            `json:"originalImageUrl,omitempty"`
	ImageUrl         string            `json:"imageUrl,omitempty"`
	ExperienceType   *string           `json:"experienceType,omitempty"`
	AttachedFileInfo AttachedFileInfo  `json:"attachedFileInfo,omitempty"`
	PreviousMessages []PreviousMessage `json:"previousMessages,omitempty"`
}

// LocationHints is a struct that represents the location hints for a message
type LocationHints struct {
	SourceType               int    `json:"SourceType,omitempty"`
	RegionType               int    `json:"RegionType,omitempty"`
	Center                   Center `json:"Center,omitempty"`
	Radius                   int    `json:"Radius,omitempty"`
	Name                     string `json:"Name,omitempty"`
	Accuracy                 int    `json:"Accuracy,omitempty"`
	FDConfidence             int    `json:"FDConfidence,omitempty"`
	CountryName              string `json:"CountryName,omitempty"`
	CountryConfidence        int    `json:"CountryConfidence,omitempty"`
	PopulatedPlaceConfidence int    `json:"PopulatedPlaceConfidence,omitempty"`
	UtcOffset                int    `json:"UtcOffset,omitempty"`
	Dma                      int    `json:"Dma,omitempty"`
}

// Center is a struct that represents the center of a location hint
type Center struct {
	Latitude  float64 `json:"Latitude,omitempty"`
	Longitude float64 `json:"Longitude,omitempty"`
}

// Participant is a struct that represents a participant of a message request
type Participant struct {
	ID string `json:"id"`
}

// GetDefaultBingRequest is a variable that holds the default message request for Bing chat
func GetDefaultBingRequest() *Request {
	return &Request{
		Arguments: []Argument{
			{
				Source: "cib",
				OptionsSets: []string{
					"nlu_direct_response_filter",
					"deepleo",
					"disable_emoji_spoken_text",
					"responsible_ai_policy_235",
					"enablemm",
					"dv3sugg",
					"iyxapbing",
					"iycapbing",
					"clgalileo",
					"gencontentv3",
					"gndbfptlw",
					"gptvnoex",
					"eredirecturl",
					"bcechat",
				},
				AllowedMessageTypes: []messagetype.MessageType{
					messagetype.ActionRequest,
					messagetype.Chat,
					messagetype.ConfirmationCard,
					messagetype.Context,
					messagetype.InternalSearchQuery,
					messagetype.InternalSearchResult,
					messagetype.Disengaged,
					messagetype.InternalLoaderMessage,
					messagetype.Progress,
					messagetype.RenderCardRequest,
					messagetype.RenderContentRequest,
					messagetype.AdsQuery,
					messagetype.SemanticSerp,
					messagetype.GenerateContentQuery,
					messagetype.SearchQuery,
					messagetype.GeneratedCode,
				},
				SliceIDs: []string{
					"ntbkcf",
					"qnacnt",
					"techpills",
					"anskeep",
					"designer2tf",
					"semseronomon-c",
					"mlchatardg",
					"cmcpupsalltf",
					"0209bicv3",
					"etlog",
					"0131gndbfpr",
					"enter4nl",
					"exptone",
				},
				Verbosity: "verbose",
				Scenario:  ScenarioSERP,
				Plugins:   []any{},
				TraceID: hex.EncodeToString(func() []byte {
					bytes := make([]byte, 16)
					rand.Read(bytes)
					return bytes
				}()),
				ConversationHistoryOptionsSets: []string{
					"threads_bce",
					"savemem",
					"uprofupd",
					"uprofgen",
				},
				IsStartOfSession: true,
				RequestID:        "",
				MessageRequest: MessageRequest{
					Locale:   "en-US",
					Market:   "en-US",
					Region:   "DE",
					Location: "lat:47.639557;long:-122.128159;re=1000m;",
					LocationHints: []LocationHints{
						{
							SourceType: 1,
							RegionType: 2,
							Center: Center{
								Latitude:  50.9275016784668,
								Longitude: 6.946300029754639,
							},
							Radius:                   24902,
							Name:                     "Germany",
							Accuracy:                 24902,
							FDConfidence:             0,
							CountryName:              "Germany",
							CountryConfidence:        9,
							PopulatedPlaceConfidence: 0,
							UtcOffset:                1,
							Dma:                      0,
						},
					},
					UserIPAddress: "2a0a:a545:98f0:0:d3e5:9cd0:f01a:666e",
					Timestamp:     time.Now().Format("2006-01-02T15:04:05-07:00"),
					Author:        "user",
					InputMethod:   "Keyboard",
					Text:          "hey",
					MessageType:   messagetype.Chat,
					RequestID:     "",
					MessageID:     "",
					Privacy:       "Internal",
				},
				Tone:           tone.Creative,
				SpokenTextMode: "None",
				ConversationID: "",
				Participant: Participant{
					ID: "",
				},
			},
		},
		InvocationID: "0",
		Target:       "chat",
		Type:         4,
	}
}

func GetDefaultCopilotRequest() *Request {
	return &Request{
		Arguments: []Argument{
			{
				Source: "cib",
				OptionsSets: []string{
					"nlu_direct_response_filter",
					"deepleo",
					"disable_emoji_spoken_text",
					"responsible_ai_policy_235",
					"enablemm",
					"dv3sugg",
					"storagev2fork",
					"papynoapi",
					"gndlogcf",
					"fluxprod",
					"revimglnk",
					"revimgsi2",
					"revimgsrc1",
					"gptvnoex",
					"bcechat",
				},
				AllowedMessageTypes: []messagetype.MessageType{
					messagetype.ActionRequest,
					messagetype.Chat,
					messagetype.ConfirmationCard,
					messagetype.Context,
					messagetype.InternalSearchQuery,
					messagetype.InternalSearchResult,
					messagetype.Disengaged,
					messagetype.InternalLoaderMessage,
					messagetype.Progress,
					messagetype.RenderCardRequest,
					messagetype.RenderContentRequest,
					messagetype.AdsQuery,
					messagetype.SemanticSerp,
					messagetype.GenerateContentQuery,
					messagetype.SearchQuery,
					messagetype.GeneratedCode,
				},
				SliceIDs: []string{
					"inlineta",
					"inlinetadisc",
					"sappbcbt",
					"bgstreamcf",
					"advperfs1",
					"designer2cf",
					"semseronomon",
					"srchqryfix",
					"mlchatardg-c",
					"cmcpupsalltf",
					"proupsallcf",
					"1215persc",
					"0209bicv3",
					"927storev2fk",
					"etlogcf",
					"0131onthdas0",
					"0208papynoa",
					"sapsgrds0",
					"1pgptwdes",
					"newzigpt",
					"1119backos",
					"enter4nlcf",
					"cacfastapis",
				},
				Verbosity: "verbose",
				Scenario:  ScenarioSERP,
				Plugins:   []any{},
				TraceID: hex.EncodeToString(func() []byte {
					bytes := make([]byte, 16)
					rand.Read(bytes)
					return bytes
				}()),
				ConversationHistoryOptionsSets: []string{
					"threads_bce",
					"savemem",
					"uprofupd",
					"uprofgen",
				},
				IsStartOfSession: true,
				RequestID:        "",
				MessageRequest: MessageRequest{
					Locale:        "en-US",
					Market:        "en-US",
					Region:        "DE",
					Location:      "lat:47.639557;long:-122.128159;re=1000m;",
					UserIPAddress: "2a0a:a545:f2e3:0:37c4:aebb:dedf:4bc",
					Timestamp:     time.Now().Format("2006-01-02T15:04:05-07:00"),
					Author:        "user",
					InputMethod:   "Keyboard",
					Text:          "hey",
					MessageType:   messagetype.Chat,
					RequestID:     "",
					MessageID:     "",
					Privacy:       "Internal",
				},
				Tone:           tone.Creative,
				SpokenTextMode: "None",
				ConversationID: "",
				Participant: Participant{
					ID: "",
				},
			},
		},
		InvocationID: "0",
		Target:       "chat",
		Type:         4,
	}
}

var toneToOptionMap map[tone.Type]string = map[tone.Type]string{
	tone.Creative: "h3imaginative",
	tone.Precise:  "h3precise",
	tone.Balanced: "galileo",
	tone.Unknown:  "harmonyv3",
}

var webSearchOption = "nosearchall"
var gpt4TurboOption = "dlgpt4t"

func (mr *Request) Setup(tone tone.Type, webSearch bool, gpt4Turbo bool, msg, msgContext, participantID, userIPAddress, conversationID, requestID string, imageUrl, originalImageUrl *string) *Request {
	// Append the options set based on the tone
	//mr.Arguments[0].OptionsSets = append(mr.Arguments[0].OptionsSets, toneToOptionMap[tone])
	mr.Arguments[0].Tone = tone

	// Append the nosearchall option if web_search is false
	if !webSearch {
		mr.Arguments[0].OptionsSets = append(mr.Arguments[0].OptionsSets, webSearchOption)
	}

	// Append the dlgpt4t option if gpt4_turbo is true
	if gpt4Turbo {
		mr.Arguments[0].OptionsSets = append(mr.Arguments[0].OptionsSets, gpt4TurboOption)
	}

	mr.Arguments[0].RequestID = requestID
	mr.Arguments[0].MessageRequest.UserIPAddress = userIPAddress
	mr.Arguments[0].MessageRequest.RequestID = requestID
	mr.Arguments[0].MessageRequest.MessageID = requestID
	mr.Arguments[0].ConversationID = conversationID
	mr.Arguments[0].Participant.ID = participantID

	mr.Arguments[0].MessageRequest.Text = msg

	// Add the context if present
	if msgContext != "" {
		mr.Arguments[0].MessageRequest.PreviousMessages = []PreviousMessage{
			{
				Author:      "user",
				Description: msgContext,
				ContextType: WebPage,
				MessageType: messagetype.Context,
				SourceName:  "Chat context",
				SourceUrl:   "http://localhost/",
				Privacy:     "Internal",
			},
		}
	}

	if imageUrl != nil {
		message := &mr.Arguments[0].MessageRequest
		message.OriginalImageUrl = *originalImageUrl
		message.ImageUrl = *imageUrl
		message.ExperienceType = nil
		message.AttachedFileInfo = AttachedFileInfo{
			FileName: nil,
			FileType: nil,
		}
	}

	return mr
}
