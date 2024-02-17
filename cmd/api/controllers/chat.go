package controllers

import (
	"context"
	"fmt"

	"git.ruekov.eu/ruakij/mcopilot-api/cmd/api/models"
	"git.ruekov.eu/ruakij/mcopilot-api/cmd/api/service"
	"github.com/gin-gonic/gin"
)

type ChatController struct {
	chatService *service.ChatService
}

func NewChatController(chatService *service.ChatService) *ChatController {
	return &ChatController{
		chatService: chatService,
	}
}

func (co ChatController) RegisterRoutes(router *gin.RouterGroup) *gin.RouterGroup {
	router.POST("/completions", co.postCompletions)

	return router
}

func (co ChatController) postCompletions(c *gin.Context) {
	var request models.ChatRequest
	if err := c.ShouldBind(&request); err != nil {
		c.JSON(400, err.Error())
		return
	}

	context, _ := context.WithCancel(c.Request.Context())

	if request.Stream {
		dataChan, _ := co.chatService.ProcessChatRequestStream(context, request)
		for {
			select {
			case <-context.Done():
				return
			case completionChunk, ok := <-dataChan:
				if ok {
					c.SSEvent("", completionChunk)
					c.Writer.Flush()
				} else {
					c.SSEvent("", "[DONE]")
					c.Writer.Flush()
					return
				}
			}
		}
	} else {
		result, err := co.chatService.ProcessChatRequest(context, request)
		if err != nil || result.ID == "" {
			c.AbortWithError(500, fmt.Errorf("no data"))
		}
		c.JSON(200, result)
	}
}
