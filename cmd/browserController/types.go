package browsercontroller

type BingChatMessageType string

const (
	Message               BingChatMessageType = ""
	CustomMessage         BingChatMessageType = "CustomMessage"
	Disengaged            BingChatMessageType = "Disengaged"
	InternalSearchQuery   BingChatMessageType = "InternalSearchQuery"
	InternalSearchResult  BingChatMessageType = "InternalSearchResult"
	AdsQuery              BingChatMessageType = "AdsQuery"
	InternalLoaderMessage BingChatMessageType = "InternalLoaderMessage"
	GenerateContentQuery  BingChatMessageType = "GenerateContentQuery"
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
	Aplology            BingChatContentOriginType = "Apology"
	JailBreakClassifier BingChatContentOriginType = "JailBreakClassifier"
	DeepLeo             BingChatContentOriginType = "DeepLeo"
)

type BingChatResponseSummary struct {
	Type uint8                    `json:"type"`
	Item BingChatResponseArgument `json:"item"`
}

type BingChatImageResponseMetadata struct {
	Title         string                                       `json:"Title"`
	ThumbnailInfo []BingChatImageResponseMetadataThumbnailInfo `json:"ThumbnailInfo"`
	CustomData    BingChatImageResponseMetadataCustomData      `json:"CustomData"`
	ContentId     string                                       `json:"ContentId"`
}

type BingChatImageResponseMetadataThumbnailInfo struct {
	ThumbnailId string
}

type BingChatImageResponseMetadataCustomData struct {
	MediaUrl string `json:"MediaUrl"`
}
