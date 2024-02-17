package service

import (
	"context"

	"git.ruekov.eu/ruakij/mcopilot-api/cmd/api/models"
	"git.ruekov.eu/ruakij/mcopilot-api/cmd/api/wrapper"
)

type ChatService struct {
	bingChatApiWrapper *wrapper.BingChatWrapper
}

func NewChatService(bingChatApiWrapper *wrapper.BingChatWrapper) *ChatService {
	return &ChatService{
		bingChatApiWrapper: bingChatApiWrapper,
	}
}

func (service *ChatService) ProcessChatRequestStream(context context.Context, request models.ChatRequest) (<-chan models.CompletionChunk, error) {
	completionChunkCh := make(chan models.CompletionChunk)

	service.bingChatApiWrapper.ProcessRequest(context, request, completionChunkCh)

	return completionChunkCh, nil
}

func (service *ChatService) ProcessChatRequest(context context.Context, request models.ChatRequest) (*models.Completion, error) {

	dataChan, err := service.ProcessChatRequestStream(context, request)
	if err != nil {
		return nil, err
	}

	var lastCompletionChunk models.CompletionChunk
	var fullText models.Message = models.Message{}
	for {
		completionChunk, ok := <-dataChan
		if !ok {
			break
		}

		lastCompletionChunk = completionChunk

		for _, choice := range completionChunk.Choices {
			fullText.Role = choice.Delta.Role
			fullText.Content += choice.Delta.Content
		}
	}

	// Build completion
	return &models.Completion{
		ID:      lastCompletionChunk.ID,
		Object:  "chat.completion",
		Created: 0,
		Model:   lastCompletionChunk.Model,
		Choices: []models.Choice{
			{
				Index: 0,
				Message: models.Message{
					Role:    fullText.Role,
					Content: fullText.Content,
				},
				FinishReason: lastCompletionChunk.Choices[len(lastCompletionChunk.Choices)-1].FinishReason,
			},
		},
	}, nil
}
