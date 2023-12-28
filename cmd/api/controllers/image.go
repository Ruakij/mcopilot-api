package controllers

import (
	"fmt"

	"git.ruekov.eu/ruakij/mcopilot-api/cmd/api/service"
	"github.com/gin-gonic/gin"
)

type ImageController struct{}

func (co ImageController) RegisterRoutes(router *gin.RouterGroup) *gin.RouterGroup {
	router = router.Group("/images")

	router.GET("/:thumbnailId", co.getImage)

	return router
}

type ThumbnailId struct {
	ThumbnailId string `uri:"thumbnailId" binding:"required"`
}

func (co ImageController) getImage(c *gin.Context) {
	var thumbnailId ThumbnailId
	err := c.ShouldBindUri(&thumbnailId)
	if err != nil {
		fmt.Println(err)
		c.JSON(400, err.Error())
		return
	}

	imageData, ok := service.ImageServiceSingleton.GetImage(thumbnailId.ThumbnailId)
	if !ok {
		c.AbortWithStatus(404)
	}

	c.Data(200, "image/jpeg", imageData)
}
