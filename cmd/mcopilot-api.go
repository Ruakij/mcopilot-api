package main

import (
	"git.ruekov.eu/ruakij/mcopilot-api/cmd/api/controllers"
	"git.ruekov.eu/ruakij/mcopilot-api/cmd/browserController"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	// Setup browser
	err := browsercontroller.Setup(&browsercontroller.LoginData{
		Email: *loginEmailFlag,
		Password: *loginPasswordFlag,
		TotpSecret: *loginTotpSecretFlag,
	})
	if err != nil {
		logger.Error.Fatalf("Error setting up BrowserController: %s", err)
	}

	// Setup Router
	router := gin.Default()

	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowCredentials = true
	router.Use(cors.New(config))
	
	v1 := router.Group("/v1")
	new(controllers.ModelController).RegisterRoutes(v1)
	new(controllers.ChatController).RegisterRoutes(v1)

	router.Run(":5000")
}
