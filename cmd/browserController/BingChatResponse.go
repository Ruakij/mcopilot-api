package browsercontroller

type BingChatMessageType string

const (
	Message               BingChatMessageType = ""
	Disengaged            BingChatMessageType = "Disengaged"
	InternalSearchQuery   BingChatMessageType = "InternalSearchQuery"
	InternalSearchResult  BingChatMessageType = "InternalSearchResult"
	AdsQuery              BingChatMessageType = "AdsQuery"
	InternalLoaderMessage BingChatMessageType = "InternalLoaderMessage"
)

type SourceAttributions struct {
	ProviderDisplayName string `json:"providerDisplayName"`
	SeeMoreUrl          string `json:"seeMoreUrl"`
	SourceType          string `json:"sourceType"`
}

type BingChatResponseMessage struct {
	Author             string                    `json:"author"`
	Text               string                    `json:"text"`
	HiddenText         string                    `json:"hiddenText"`
	MessageType        BingChatMessageType       `json:"messageType"`
	SourceAttributions []SourceAttributions      `json:"sourceAttributions,omitempty"`
	Offence            string                    `json:"offence"`
	ContentOrigin      BingChatContentOriginType `json:"contentOrigin"`
}
type BingChatResponseArgument struct {
	Messages  []BingChatResponseMessage `json:"messages"`
	RequestId string                    `json:"requestId"`
}
type BingChatResponse struct {
	Type      uint8                      `json:"type"`
	Target    string                     `json:"target"`
	Arguments []BingChatResponseArgument `json:"arguments"`
}

type BingChatContentOriginType string

const (
	Unknown             BingChatContentOriginType = ""
	Aplology            BingChatContentOriginType = "Aplology"
	JailBreakClassifier BingChatContentOriginType = "JailBreakClassifier"
)

type BingChatResponseSummary struct {
	Type uint8                    `json:"type"`
	Item BingChatResponseArgument `json:"item"`
}
