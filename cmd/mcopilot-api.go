package main

import (
	"os"
	"strconv"

	"git.ruekov.eu/ruakij/mcopilot-api/cmd/api/controllers"
	"git.ruekov.eu/ruakij/mcopilot-api/cmd/api/service"
	"git.ruekov.eu/ruakij/mcopilot-api/cmd/browserController"
	"git.ruekov.eu/ruakij/mcopilot-api/lib/environmentchecks"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/vence722/gin-middleware-bearer-token"
)

var envRequired = []string{
	"LOGIN_EMAIL",
	"LOGIN_PASSWORD",
}
var envDefaults = map[string]string{
	"LOGIN_TOTP_SECRET": "",

	"WORKER_COUNT":    "1",
	"BASE_URL":        "http://127.0.0.1:5000",
	"BROWSERDATA_DIR": "/data/browser",
	"IMAGE_DIR":       "/data/images",

	"AUTH_TOKEN": "MCOPILOT_API_SECRET_TOKEN",
}

func main() {
	// Environment-vars
	err := environmentchecks.HandleRequired(envRequired)
	if err != nil {
		logger.Error.Fatal(err)
	}
	environmentchecks.HandleDefaults(envDefaults)

	// Setup services
	imageService := new(service.ImageService).Init(os.Getenv("IMAGE_DIR"))

	// Setup browser
	workerCount, err := strconv.Atoi(os.Getenv("WORKER_COUNT"))
	if err != nil {
		logger.Error.Fatal(err)
	}
	err = browsercontroller.Setup(&browsercontroller.LoginData{
		Email:      os.Getenv("LOGIN_EMAIL"),
		Password:   os.Getenv("LOGIN_PASSWORD"),
		TotpSecret: os.Getenv("LOGIN_TOTP_SECRET"),
	},
		workerCount,
		os.Getenv("BROWSERDATA_DIR"),
		os.Getenv("BASE_URL"),
		func(key string, data []byte) {
			imageService.PutImage(key, data)
		},
	)
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
	new(controllers.ChatController).RegisterRoutes(v1).Use(
		bearertoken.MiddlewareWithStaticToken(os.Getenv("AUTH_TOKEN")),
	)
	new(controllers.ImageController).RegisterRoutes(v1)

	router.Run(":5000")
}
