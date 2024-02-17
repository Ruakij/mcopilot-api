package controllers

import (
	"github.com/gin-gonic/gin"
)

type Model struct {
	ID       string `json:"id"`
	Object   string `json:"object"`
	Created  int32  `json:"created"`
	Owned_by string `json:"owned_by"`
}
type ModelList struct {
	Object string  `json:"object"`
	Data   []Model `json:"data"`
}

type ModelController struct{}

func (co ModelController) RegisterRoutes(router *gin.RouterGroup) *gin.RouterGroup {
	//router = router.Group("/models")

	router.GET("/models", co.getModels)

	return router
}

var modelList = ModelList{
	Object: "list",
	Data: []Model{
		{
			ID:       "Creative",
			Created:  0,
			Owned_by: "system",
			Object:   "model",
		},
		{
			ID:       "Balanced",
			Created:  0,
			Owned_by: "system",
			Object:   "model",
		},
		{
			ID:       "Precise",
			Created:  0,
			Owned_by: "system",
			Object:   "model",
		},
	},
}

/*
{
  "object": "list",
  "data": [
    {
      "id": "gpt-4-vision-preview",
      "object": "model",
      "created": 1698894917,
      "owned_by": "system"
    },
    ...
*/

func (co ModelController) getModels(c *gin.Context) {
	c.JSON(200, modelList)
}
