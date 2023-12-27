package controllers

import (
	"context"
	"fmt"

	"git.ruekov.eu/ruakij/mcopilot-api/cmd/api/models"
	"git.ruekov.eu/ruakij/mcopilot-api/cmd/api/service"
	"github.com/gin-gonic/gin"
)

type ChatController struct{}

func (co ChatController) RegisterRoutes(router *gin.RouterGroup) {
	router = router.Group("/chat")

	router.POST("/completions", co.postCompletions)
}

func (co ChatController) postCompletions(c *gin.Context) {
	var request models.ChatRequest
	if err := c.ShouldBind(&request); err != nil {
		c.JSON(400, err.Error())
		return
	}

	dataChan := make(chan models.CompletionChunk, 200)
	resultChan := make(chan models.Completion)

	context, cancel := context.WithCancel(context.Background())

	ChatService.ProcessChatRequest(context, request, dataChan, resultChan)

	if request.Stream {
		for {
			select {
			case <-c.Request.Context().Done():
				cancel()
				return
			case completionChunk, ok := <-dataChan:
				if ok {
					c.SSEvent("", completionChunk)
					c.Writer.Flush()
				} else {
					c.SSEvent("", "[DONE]")
					c.Writer.Flush()
					
					cancel()
					return
				}
			}
		}
	} else {
		result := <-resultChan
		if result.ID == ""{
			c.AbortWithError(500, fmt.Errorf("no data"))
		}
		c.JSON(200, result)

		cancel()
	}
}
