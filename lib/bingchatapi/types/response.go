package types

import (
	"git.ruekov.eu/ruakij/mcopilot-api/lib/bingchatapi/types/contentorigin"
	"git.ruekov.eu/ruakij/mcopilot-api/lib/bingchatapi/types/messagetype"
)

type SourceAttributions struct {
	ProviderDisplayName string `json:"providerDisplayName"`
	SeeMoreUrl          string `json:"seeMoreUrl"`
	SourceType          string `json:"sourceType"`
}

type BingChatResponseArgumentResult struct {
	Value          BingChatResponseArgumentResultValue `json:"value"`
	Message        string                              `json:"message"`
	Error          string                              `json:"error"`
	ServiceVersion string                              `json:"serviceVersion"`
}
type BingChatResponseArgumentResultValue string

const (
	CaptchaChallenge BingChatResponseArgumentResultValue = "CaptchaChallenge"
	Success          BingChatResponseArgumentResultValue = "Success"
)

type BingChatResponseMessage struct {
	Author             string                  `json:"author"`
	Text               string                  `json:"text"`
	HiddenText         string                  `json:"hiddenText"`
	MessageType        messagetype.MessageType `json:"messageType"`
	SourceAttributions []SourceAttributions    `json:"sourceAttributions,omitempty"`
	Offence            string                  `json:"offence"`
	ContentOrigin      contentorigin.Type      `json:"contentOrigin"`
}
type BingChatResponseArgument struct {
	Messages  []BingChatResponseMessage      `json:"messages"`
	RequestId string                         `json:"requestId"`
	Result    BingChatResponseArgumentResult `json:"result"`
}

type BingChatResponse struct {
}

// General
type BingChatResponseBasic struct {
	Type uint8 `json:"type"`
}

// Type 1
type BingChatResponseNormal struct {
	Type      uint8                      `json:"type"`
	Target    string                     `json:"target"`
	Arguments []BingChatResponseArgument `json:"arguments"`
}

// Type 2
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
