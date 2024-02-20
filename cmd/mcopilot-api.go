package main

import (
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"git.ruekov.eu/ruakij/mcopilot-api/cmd/api/controllers"
	"git.ruekov.eu/ruakij/mcopilot-api/cmd/api/logger"
	"git.ruekov.eu/ruakij/mcopilot-api/cmd/api/service"
	"git.ruekov.eu/ruakij/mcopilot-api/cmd/api/wrapper"
	"git.ruekov.eu/ruakij/mcopilot-api/lib/bingchatapi/browser"
	"git.ruekov.eu/ruakij/mcopilot-api/lib/bingchatapi/reng"
	"git.ruekov.eu/ruakij/mcopilot-api/lib/bingchatapi/types/tone"
	"git.ruekov.eu/ruakij/mcopilot-api/lib/environmentchecks"
	"git.ruekov.eu/ruakij/mcopilot-api/lib/httpmisc"
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

	"BROWSER_HEADLESS":           "true",
	"BROWSER_REMOTE_CONTROL_URL": "",
	"BROWSER_ARGS":               "",

	"SESSION_SOFT_TIMEOUT": "30m",
	"SESSION_HARD_TIMEOUT": "60m",
	"SESSION_MAX_COUNT":    "10",

	"AUTH_COOKIES": "",
}

func main() {
	// Environment-vars
	err := environmentchecks.HandleRequired(envRequired)
	if err != nil {
		logger.Error.Fatal(err)
	}
	environmentchecks.HandleDefaults(envDefaults)

	sessionSoftTimeout, err := time.ParseDuration(os.Getenv("SESSION_SOFT_TIMEOUT"))
	if err != nil {
		logger.Error.Fatalf("Error parsing SESSION_SOFT_TIMEOUT: %s", err)
	}

	sessionHardTimeout, err := time.ParseDuration(os.Getenv("SESSION_HARD_TIMEOUT"))
	if err != nil {
		logger.Error.Fatalf("Error parsing SESSION_HARD_TIMEOUT: %s", err)
	}

	// Setup services
	imageService := new(service.ImageService).Init(os.Getenv("IMAGE_DIR"), sessionSoftTimeout)
	imageService = imageService

	// Setup browser
	workerCount, err := strconv.Atoi(os.Getenv("WORKER_COUNT"))
	if err != nil {
		logger.Error.Fatal(err)
	}
	logger.Info.Println("Setup browser..")
	headless, err := strconv.ParseBool(os.Getenv("BROWSER_HEADLESS"))
	if err != nil {
		logger.Error.Fatal(err)
	}

	browserArgs := make(map[string][]string)
	if os.Getenv("BROWSER_ARGS") != "" {
		splitChars := ";"
		if strings.Contains(os.Getenv("BROWSER_ARGS"), "\n") {
			splitChars = "\n"
		}
		args := strings.Split(os.Getenv("BROWSER_ARGS"), splitChars)

		for _, arg := range args {
			if arg == "" {
				continue
			}
			if strings.Contains(arg, "=") {
				keyValues := strings.SplitN(arg, "=", 2)
				browserArgs[keyValues[0]] = keyValues[1:]
			} else {
				browserArgs[arg] = []string{}
			}
		}
	}
	err = browser.Setup(
		headless,
		os.Getenv("BROWSER_REMOTE_CONTROL_URL"),
		browserArgs,
		&browser.LoginData{
			Email:      os.Getenv("LOGIN_EMAIL"),
			Password:   os.Getenv("LOGIN_PASSWORD"),
			TotpSecret: os.Getenv("LOGIN_TOTP_SECRET"),
		},
		0,
		os.Getenv("BROWSERDATA_DIR"),
		os.Getenv("BASE_URL"),
		func(key string, data []byte) {
			imageService.PutImage(key, data)
		},
	)
	if err != nil {
		logger.Error.Fatalf("Error setting up BrowserController: %s", err)
	}

	cookies := make(map[string]string, 1)
	cookieString := os.Getenv("AUTH_COOKIES")
	cookieList := strings.Split(cookieString, "; ")
	for _, singleCookieString := range cookieList {
		singleCookieKeyValue := strings.SplitN(singleCookieString, "=", 2)
		if len(singleCookieKeyValue) < 2 {
			continue
		}
		cookies[singleCookieKeyValue[0]] = singleCookieKeyValue[1]
	}

	// Setup api
	api := reng.NewRengApi(workerCount, cookies)
	apiWrapper := wrapper.NewBingChatApiWrapper(api)

	page, err := browser.GetNewReadyPage()
	if err != nil {
		logger.Error.Fatalf("Failed getting page in main: %s", err)
	}
	pageMutex := new(sync.Mutex)
	var pageTimer *time.Timer
	var pageReopenTimer *time.Timer
	if sessionSoftTimeout > 0 {
		pageTimer = time.AfterFunc(sessionSoftTimeout, func() {
			pageMutex.Lock()
			defer pageMutex.Unlock()

			if pageReopenTimer != nil {
				pageReopenTimer.Stop()
			}

			logger.Info.Printf("Closing the page after %s", sessionSoftTimeout.String())
			if page != nil {
				page.Close()
				page = nil
			}
		})
	}
	// Add a new timer to close and reopen the page after 60 minutes
	if sessionHardTimeout > 0 {
		pageReopenTimer = time.AfterFunc(sessionHardTimeout, func() {
			if page != nil {
				logger.Info.Printf("Opening a new page after %s minutes\n", sessionHardTimeout)

				newPage, err := browser.GetNewReadyPage()
				if err != nil {
					logger.Error.Fatalf("Failed getting page: %s", err.Error())
				}

				pageMutex.Lock()
				logger.Info.Println("Setting new page as active and closing old page")
				page.Close()
				page = newPage
				pageReopenTimer.Reset(sessionHardTimeout)
				pageMutex.Unlock()
			}
		})
	}
	api.SetHooks(&reng.Hooks{
		CreateConversation: func(session *httpmisc.HttpClientSession, tone tone.Type, imageData *string) (conversation *reng.Conversation, err error) {
			pageMutex.Lock()
			defer pageMutex.Unlock()

			if pageTimer != nil {
				pageTimer.Reset(sessionSoftTimeout)
			}

			if page == nil {
				logger.Info.Println("Getting new page..")
				page, err = browser.GetNewReadyPage()
				if err != nil {
					logger.Error.Fatalf("Failed getting page: %s", err.Error())
				}
				if pageReopenTimer != nil {
					pageReopenTimer.Reset(sessionHardTimeout)
				}
			}

			logger.Info.Println("Getting conversation via browser..")
			conversationId, clientId, signature, err := browser.GetManualConversation(page)

			if err != nil {
				logger.Info.Println("Failed getting conversation via browser.. reopening page to retry fresh")
				page.Close()
				page, err = browser.GetNewReadyPage()
				if err != nil {
					logger.Error.Fatalf("Failed getting page: %s", err.Error())
				}
				if pageTimer != nil {
					pageTimer.Reset(sessionSoftTimeout)
				}
				if pageReopenTimer != nil {
					pageReopenTimer.Reset(sessionHardTimeout)
				}

				return
			}

			networkCookies := page.MustCookies()
			session.DefaultCookies = make(map[string]string)
			for _, networkCookie := range networkCookies {
				session.DefaultCookies[networkCookie.Name] = networkCookie.Value
			}

			conversation = &reng.Conversation{
				ConversationId:        conversationId,
				ClientId:              clientId,
				ConversationSignature: signature,
			}

			return
		},
	})

	// Start api
	apiWrapper.Init()

	// Setup services
	chatService := service.NewChatService(apiWrapper)

	// Setup Router
	logger.Info.Println("Setup router..")
	router := gin.Default()

	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowCredentials = true
	config.AllowHeaders = append(config.AllowHeaders, "Authorization")
	//router.Use(cors.New(config))
	router.Use(CORSMiddleware())

	v1 := router.Group("/v1")
	new(controllers.ModelController).RegisterRoutes(v1)
	chatRouter := v1.Group("/chat")
	chatRouter.Use(
		bearertoken.MiddlewareWithStaticToken(os.Getenv("AUTH_TOKEN")),
	)
	controllers.NewChatController(chatService).RegisterRoutes(chatRouter)

	new(controllers.ImageController).RegisterRoutes(v1)

	logger.Info.Println("Start router..")
	router.SetTrustedProxies([]string{"10.0.0.0/8"})
	router.Run(":5000")
}

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Header("Access-Control-Allow-Methods", "POST,HEAD,PATCH, OPTIONS, GET, PUT")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
