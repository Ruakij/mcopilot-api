package browsercontroller

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/pquerna/otp/totp"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"

	//"github.com/go-rod/stealth"

	"encoding/json"
)

type LoginData struct {
	Email      string
	Password   string
	TotpSecret string
}

var loginData *LoginData

var BASE_URL string

var PutImageHook func(string, []byte)

func Setup(setupLoginData *LoginData, workerCount int, browserDataDir string, baseUrl string, putImageHook func(string, []byte)) (err error) {
	loginData = setupLoginData
	BASE_URL = baseUrl
	PutImageHook = putImageHook

	err = setupRod(browserDataDir)
	if err != nil {
		return
	}

	for i := 0; i < workerCount; i++ {
		go processRequests()
	}

	return
}

type WorkItem struct {
	Page         *rod.Page
	Context      context.Context
	Input        string
	OutputStream chan<- BingChatResponse
}

var workQueue chan *WorkItem = make(chan *WorkItem)

// Place request into queue, blocks until work is processing
func ProcessChatRequestWithPage(context context.Context, page *rod.Page, text string, streamChan chan<- BingChatResponse) {
	work := &WorkItem{
		Page:         page,
		Context:      context,
		Input:        text,
		OutputStream: streamChan,
	}

	workQueue <- work
}
func ProcessChatRequest(work *WorkItem) {
	workQueue <- work
}

var browser *rod.Browser

func setupRod(browserDataDir string) (err error) {
	// Connect to the WebDriver instance running locally.

	launcher := launcher.New()
	launcher.UserDataDir(browserDataDir)
	curl, err := launcher.Launch()
	if err != nil {
		return err
	}
	browser = rod.New().ControlURL(curl)
	err = browser.Connect()
	if err != nil {
		return err
	}

	/*
		caps := selenium.Capabilities{
			"browserName": "Chrome",
		}
		selenium.Capabilities.AddChrome(caps, chrome.Capabilities{
			Args: []string{
				"--disable-gpu",
				"--headless",
				"--no-sandbox",
				"--disable-sync",
				"--no-first-run",
				"--autoplay-policy=no-user-gesture-required",
				"--use-fake-ui-for-media-stream",
				"--use-fake-device-for-media-stream",
			},
		})
	*/
	return
}

func isChannelClosed(channel chan any) bool {
	select {
	case _, ok := <-channel:
		return !ok
	default:
		return false
	}
}

// handleWorkItem handles a single work item by sending the input to the page and receiving the output
func handleWorkItem(work WorkItem) error {
	page := work.Page

	inputElement, err := ElementImmediateRecursive(page, "#searchbox")
	if err != nil {
		return err
	}

	waitChan := make(chan BingChatMessageType)
	waitCloseChan := make(chan interface{})

	var followUpReason BingChatMessageType = Message
	go page.EachEvent(func(e *proto.NetworkWebSocketCreated) {
		waitChan <- ""
	}, func(e *proto.NetworkWebSocketFrameReceived) {
		//if e.Response.Opcode != 1 { return }

		for _, payload := range strings.Split(e.Response.PayloadData, "\x1e") {
			if len(payload) == 0 {
				continue
			}

			var bingChatResponse BingChatResponse
			err := json.Unmarshal([]byte(payload), &bingChatResponse)
			if err != nil {
				fmt.Println(err)
				continue
			}

			switch bingChatResponse.Type {
			case 1:
				if len(bingChatResponse.Arguments) > 0 && len(bingChatResponse.Arguments[len(bingChatResponse.Arguments)-1].Messages) > 0 {
					if isChannelClosed(waitCloseChan) {
						return
					}
					work.OutputStream <- bingChatResponse
				}
			case 2:
				var bingChatResponseSummary BingChatResponseSummary
				json.Unmarshal([]byte(payload), &bingChatResponseSummary)
				if err != nil {
					continue
				}

				// Enter summary-data to stream
				if isChannelClosed(waitCloseChan) {
					return
				}
				work.OutputStream <- BingChatResponse{
					Type:      bingChatResponseSummary.Type,
					Arguments: []BingChatResponseArgument{bingChatResponseSummary.Item},
				}

				// Check for follow-ups
				for _, message := range bingChatResponseSummary.Item.Messages {
					switch message.MessageType {
					case GenerateContentQuery:
						followUpReason = message.MessageType
					}
				}

				// Tell others stream is over
				waitChan <- followUpReason
			}
		}
	}, func(e *proto.NetworkWebSocketClosed) {
		waitChan <- followUpReason
	})()

	// Hijack for catching additional requests
	hijackRouter := page.HijackRequests()
	imageResponses := []BingChatImageResponseMetadata{}
	imageCountSeen := 0
	hijackRouter.MustAdd("https://th.bing.com/th/id/*", func(ctx *rod.Hijack) {
		ctx.MustLoadResponse()

		if ctx.Request.Method() != "GET" {
			return
		}

		// Extract image-id from url
		UrlData := regexp.MustCompile(`\/([^\/]+?)($|\?)`).FindStringSubmatch(ctx.Request.URL().Path)
		ThumbnailId := UrlData[1]
		/*var imageMetadata BingChatImageResponseMetadata
		for _, imageResponse := range imageResponses {
			if imageResponse.ThumbnailInfo[0].ThumbnailId == ThumbnailId {
				imageMetadata = imageResponse
				break
			}
		}*/

		//ctx.Request.URL().Query().Del("")

		imageData := ctx.Response.Payload().Body

		if len(imageData) > 0 {
			// Store data
			PutImageHook(ThumbnailId, imageData)

			// Build fake response to display image
			work.OutputStream <- BingChatResponse{
				Type: 1,
				Arguments: []BingChatResponseArgument{
					{
						Messages: []BingChatResponseMessage{
							{
								MessageType: CustomMessage,
								Author:      "bot",
								Text:        fmt.Sprintf("![](%s/v1/images/%s) ", BASE_URL, ThumbnailId),
							},
						},
					},
				},
			}
		}

		imageCountSeen++
		if len(imageResponses) == imageCountSeen {
			waitChan <- ""
		}
	})
	hijackRouter.MustAdd("https://copilot.microsoft.com/images/create/async/results*", func(ctx *rod.Hijack) {
		ctx.MustLoadResponse()
		responseBody := ctx.Response.Payload().Body
		if len(responseBody) == 0 {
			return
		}
		responseBodyStr := string(responseBody)

		// Extract image-info
		imageDataAll := regexp.MustCompile(` ?m="(\{.+\})"`).FindAllStringSubmatch(responseBodyStr, -1)
		for _, imageDataSlice := range imageDataAll {
			imageData := imageDataSlice[1]
			imageData = strings.ReplaceAll(imageData, "&quot;", `"`)
			imageData = strings.ReplaceAll(imageData, "&amp;", `&`)
			imageData = strings.ReplaceAll(imageData, `\"`, `"`)
			imageData = regexp.MustCompile(`"CustomData":\s*"(\{.+?\})"`).ReplaceAllString(imageData, `"CustomData": $1`)

			var metadata BingChatImageResponseMetadata
			err := json.Unmarshal([]byte(imageData), &metadata)
			if err != nil {
				fmt.Println(err)
				continue
			}

			imageResponses = append(imageResponses, metadata)
		}

		responseBodyStr = regexp.MustCompile(`w=\d+(&amp;)?`).ReplaceAllString(responseBodyStr, "")
		responseBodyStr = regexp.MustCompile(`h=\d+(&amp;)?`).ReplaceAllString(responseBodyStr, "")
		responseBodyStr = regexp.MustCompile(`c=\d+(&amp;)?`).ReplaceAllString(responseBodyStr, "")
		//ctx.Response = ctx.Response.SetBody(responseBodyStr)

		if len(imageResponses) > 0 {
			work.OutputStream <- BingChatResponse{
				Type: 1,
				Arguments: []BingChatResponseArgument{
					{
						Messages: []BingChatResponseMessage{
							{
								MessageType: CustomMessage,
								Author:      "bot",
								Text:        "\n\nResult:\n",
							},
						},
					},
				},
			}
		} else {
			work.OutputStream <- BingChatResponse{
				Type: 1,
				Arguments: []BingChatResponseArgument{
					{
						Messages: []BingChatResponseMessage{
							{
								MessageType: CustomMessage,
								Author:      "bot",
								Text:        "\n\n[SYSTEM] Mhh.. something went wrong.",
							},
						},
					},
				},
			}
			waitChan <- ""
		}

		time.Sleep(time.Second * 10)
		if !isChannelClosed(waitCloseChan) {
			work.OutputStream <- BingChatResponse{
				Type: 1,
				Arguments: []BingChatResponseArgument{
					{
						Messages: []BingChatResponseMessage{
							{
								MessageType: CustomMessage,
								Author:      "bot",
								Text:        "\n\n[SYSTEM] Mhh.. that took too long.",
							},
						},
					},
				},
			}
			waitChan <- ""
		}
	})
	go hijackRouter.Run()

	// Type in
	inputElement.Eval("(input) => this.value = input", work.Input)
	// Trigger input event
	inputElement.Eval(` () => {
		var event = new Event('input', {
			bubbles: true,
			cancelable: true,
		});
		
		this.dispatchEvent(event);
	}
	`)
	//inputElement.MustInput(work.input)

	sendBtn, err := ElementImmediateRecursive(page, ".submit button")
	if err != nil {
		return err
	}
	work.Page.Activate()
	time.Sleep(time.Millisecond * 50)
	sendBtn.MustEval("() => this.click()")

	// Wait for stream to start
	streamStarted := false
	select {
	case <-waitChan:
		streamStarted = true
		fmt.Println("Stream started")
	case <-time.After(time.Second * 10):
		if !streamStarted {
			fmt.Println("Stream not started after 10s")
			close(waitCloseChan)
			close(work.OutputStream)
			return nil
		}
	case <-work.Context.Done():
		// Request was cancelled
		close(waitCloseChan)
		work.Page.Close() // TODO: Properly cancel BingChat
		close(work.OutputStream)
		return nil
	}

	for !isChannelClosed(waitCloseChan) {
		select {
		case followUpReason := <-waitChan:
			switch followUpReason {
			case GenerateContentQuery:
				// Dont stop yet as more data will be sent from other locations
				continue
			}

			close(waitCloseChan)
			close(work.OutputStream)
		case <-work.Context.Done():
			// Request was cancelled
			close(waitCloseChan)
			work.Page.Close() // TODO: Properly cancel BingChat
			close(work.OutputStream)
		}
	}
	hijackRouter.Stop()
	return nil
}

// processRequests processes work items from the work queue
func processRequests() {
	var curPage *rod.Page
	for {
		for curPage == nil {
			curPage, _ = getNewReadyPage()
			if curPage == nil {
				time.Sleep(time.Second * 30)
			}
		}
		fmt.Println("Ready!")

		var work *WorkItem = <-workQueue
		if work.Page == nil {
			work.Page = curPage
			curPage = nil
		}

		err := handleWorkItem(*work)
		if err != nil {
			fmt.Println(err)
		}
	}
}

const targetUrl string = "https://copilot.microsoft.com/"

func getNewReadyPage() (page *rod.Page, err error) {
	//page = stealth.MustPage(browser)
	page, err = browser.Page(proto.TargetCreateTarget{URL: targetUrl})
	if err != nil {
		return
	}

	page.MustEvaluate(&rod.EvalOptions{
		JS: `() => {
		function overrideFocusDetection() {
			// Store the original methods in variables
			var originalHidden = document.hidden;
			var originalVisibilityState = document.visibilityState;
			var originalVisibilityChange = document.onvisibilitychange;
			var originalBlur = window.onblur;
			var originalFocus = window.onfocus;

			// Define new methods that always return true for focus
			document.hidden = false;
			document.visibilityState = "visible";
			document.onvisibilitychange = function() {};
			window.onblur = function() {};
			window.onfocus = function() {};

			// Return a function that restores the original methods
			return function restoreFocusDetection() {
				document.hidden = originalHidden;
				document.visibilityState = originalVisibilityState;
				document.onvisibilitychange = originalVisibilityChange;
				window.onblur = originalBlur;
				window.onfocus = originalFocus;
			};
		}

		// Call the function and store the restore function in a variable
		var restore = overrideFocusDetection();
	}`,
	})

	//page.Navigate(targetUrl)
	page.MustWaitLoad()

	if needsLogin(page) {
		initiateLogin(page)
		page.Reload()

		err = page.WaitStable(time.Duration(3) * time.Second)
		if err != nil {
			return
		}

		if needsLogin(page) {
			return page, fmt.Errorf("login ran, but still need login")
		}
	}

	inputElement, err := ElementImmediateRecursive(page, "#searchbox")
	if err != nil {
		// Find potential errors
		return nil, fmt.Errorf("textbox not found: %e", err)
	}

	// Select creative
	creativeBtn, _ := ElementImmediateRecursive(page, "button.tone-creative")
	creativeBtn.Eval("() => this.click()")

	time.Sleep(time.Duration(1) * time.Second)
	inputElement.Eval("() => this.removeAttribute('maxlength')")

	// Setup Copilot-settings
	page.MustEval(`() => {
		CIB.config.messaging.enableSyntheticStreaming = false;
		CIB.config.messaging.streamSyntheticTextResponses = false;
	}`)

	return
}

func needsLogin(page *rod.Page) bool {
	action, _, _ := getLoginButton(page)
	return action != nil
}

type Action struct {
	Selector string
	Next     *Action
}

var loginActions []Action = []Action{
	{
		Selector: "a[aria-label='Sign in']",
	},
	{
		Selector: "input[value='Sign in']",
		Next: &Action{
			Selector: ".id_accountItem:nth-child(2) a",
		},
	},
}

func getLoginButton(page *rod.Page) (*Action, *rod.Element, error) {
	for _, action := range loginActions {
		elm, err := ElementImmediate(page, action.Selector)
		if elm != nil {
			visible, _ := elm.Visible()
			if visible {
				return &action, elm, err
			}
		}
	}

	return nil, nil, fmt.Errorf("no login-button found or visible")
}
func initiateLogin(page *rod.Page) (err error) {
	currAction, elm, err := getLoginButton(page)
	if err != nil {
		return
	}

	log.Println("Clicking login button") // Add log message
	elm.Click(proto.InputMouseButtonLeft, 1)
	time.Sleep(time.Duration(1) * time.Second)
	for currAction.Next != nil {
		currAction = currAction.Next

		elm, err = ElementImmediate(page, currAction.Selector)
		if err != nil {
			return
		}

		log.Printf("Clicking %s\n", currAction.Selector) // Add log message
		elm.Click(proto.InputMouseButtonLeft, 1)
		time.Sleep(time.Duration(1) * time.Second)
	}

	newPage := GetFirstNotEquals(browser.MustPages(), page)
	if newPage == nil {
		return fmt.Errorf("no login-page opened")
	}

	newPage.WaitStable(time.Duration(2) * time.Second)

	emailField, err := ElementImmediate(newPage, "input[type='email']")
	if err != nil {
		return fmt.Errorf("no email-field found")
	}
	log.Println("Entering email") // Add log message
	emailField.MustInput(loginData.Email)
	newPage.Keyboard.Type(input.Enter)

	newPage.WaitStable(time.Duration(2) * time.Second)

	passwordField, err := ElementImmediate(newPage, "input[type='password']")
	if err != nil {
		return fmt.Errorf("no password-field found")
	}
	log.Println("Entering password") // Add log message
	passwordField.MustInput(loginData.Password)
	newPage.Keyboard.Type(input.Enter)

	newPage.WaitStable(time.Duration(2) * time.Second)

	totpField, err := ElementImmediate(newPage, "input[placeholder='Code']")
	if err != nil {
		return fmt.Errorf("no totp-field found")
	}
	totpCode, _ := totp.GenerateCode(loginData.TotpSecret, time.Now())
	log.Println("Entering totp code") // Add log message
	totpField.MustInput(totpCode)
	newPage.Keyboard.Type(input.Enter)

	newPage.WaitStable(time.Duration(2) * time.Second)

	// Check for "Stay signed in" dialog
	yesField, err := ElementImmediate(newPage, "input[value='Yes']")
	if err == nil {
		log.Println("Clicking yes to stay signed in") // Add log message
		yesField.Click(proto.InputMouseButtonLeft, 1)
	}

	newPage.WaitStable(time.Duration(3) * time.Second)

	log.Println("Login successful") // Add log message
	newPage.Close()

	return
}

func GetFirstNotEquals(pages rod.Pages, page *rod.Page) *rod.Page {
	for _, p := range pages {
		if p != page {
			return p
		}
	}

	return nil
}

func ElementImmediate(page *rod.Page, selector string) (*rod.Element, error) {
	elms := page.MustElements(selector)
	if elms.Empty() {
		return nil, fmt.Errorf("no element on page")
	}

	return elms.First(), nil
}

func ElementImmediateRecursive(page *rod.Page, selector string) (*rod.Element, error) {
	elms := page.MustElementsByJS(`() => {
		function recursiveQuerySelector(selector) {
			let element = null;
			function findElement(root) {
				if (element) return; // If element is found, stop searching
				for (const node of root.querySelectorAll("*")) {
					if (node.matches(selector)) {
						element = node;
						return;
					}
					if (node.shadowRoot) {
						// Look for elements in the current root
						findElement(node.shadowRoot);
					}
				}
			}
			findElement(document);
			return element;
		}

		return [recursiveQuerySelector('` + selector + `')]
	}
	`)

	if elms.Empty() {
		return nil, fmt.Errorf("no element on page")
	}

	return elms.First(), nil
}
