package controllers

import "github.com/gin-gonic/gin"

type Model struct {
	ID       string `json:"id"`
	Object   string `json:"object"`
	Created  int32  `json:"created"`
	Owned_by string `json:"owned_by"`
}

type ModelController struct{}

func (co ModelController) RegisterRoutes(router *gin.RouterGroup) *gin.RouterGroup {
	router = router.Group("/models")

	router.GET("/", co.getModels)

    return router;
}

var modelList = []Model{
    {
		ID: "Creative",
        Created: 0,
        Owned_by: "",
        Object: "model",
    },
    {
        ID: "Balanced",
        Created: 0,
        Owned_by: "",
		Object: "model",
    },
	{
        ID: "Precise",
        Created: 0,
        Owned_by: "",
		Object: "model",
    },
}

func (co ModelController) getModels(c *gin.Context){
	c.JSON(200, modelList)
}
