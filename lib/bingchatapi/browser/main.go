package browser

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"git.ruekov.eu/ruakij/mcopilot-api/cmd/api/logger"
	"github.com/pquerna/otp/totp"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/devices"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/launcher/flags"
	"github.com/go-rod/rod/lib/proto"
	//"github.com/go-rod/stealth"
)

type LoginData struct {
	Email      string
	Password   string
	TotpSecret string
}

var loginData *LoginData

var BASE_URL string

var PutImageHook func(string, []byte)

func Setup(headless bool, remoteControlUrl string, browserArgs map[string][]string, setupLoginData *LoginData, workerCount int, browserDataDir string, baseUrl string, putImageHook func(string, []byte)) (err error) {
	loginData = setupLoginData
	BASE_URL = baseUrl
	PutImageHook = putImageHook

	err = setupRod(headless, remoteControlUrl, browserDataDir, browserArgs)
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
	OutputStream chan<- []byte
}

var workQueue chan *WorkItem = make(chan *WorkItem)

// Place request into queue, blocks until work is processing
func ProcessChatRequestWithPage(context context.Context, page *rod.Page, text string, streamChan chan<- []byte) {
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

func setupRod(headless bool, remoteControlUrl string, browserDataDir string, browserArgs map[string][]string) (err error) {
	// If no remote is specified, start a browser
	if remoteControlUrl == "" {

		l := launcher.New()

		l.UserDataDir(browserDataDir).Headless(headless)

		//if !headless {
		//	l.XVFB("--server-num=1", "--server-args=-screen 0 1600x900x16")
		//}

		for key, value := range browserArgs {
			flag := flags.Flag(key)
			flag.Check()

			if len(value) == 0 {
				l.Set(flag)
				logger.Info.Printf("Setting arg %s", key)
			} else {
				logger.Info.Printf("Setting arg %s=%s", key, append(value, ","))
			}
		}

		if !headless {
			//l.Set("--disable-gpu")
			//l.Set("--no-sandbox")
			//l.Set("--disable-sync")
			l.Set("--no-first-run")
			l.Set("--use-fake-ui-for-media-stream")
			l.Set("--use-fake-device-for-media-stream")
			//l.Set("--disable-dev-shm-usage")
			//l.Set("--fullscreen")
		}

		logger.Info.Println("Launch browser..")
		remoteControlUrl, err = l.Launch()
		if err != nil {
			return err
		}
	}

	browser = rod.New().ControlURL(remoteControlUrl)
	// Connect
	logger.Info.Println("Connect to browser..")
	browser = browser.MustConnect()

	return
}

const targetUrl string = "https://copilot.microsoft.com/"

//const targetUrl string = "https://www.bing.com/search?q=Bing+AI&showconv=1"

func GetNewReadyPage() (page *rod.Page, err error) {
	logger.Info.Println("Opening page")
	//page = stealth.MustPage(browser)
	page, err = browser.Page(proto.TargetCreateTarget{})
	if err != nil {
		return
	}

	page.Emulate(devices.LaptopWithHiDPIScreen.Landscape())
	page.SetUserAgent(&proto.NetworkSetUserAgentOverride{
		UserAgent:      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.2210.91",
		Platform:       "Win32",
		AcceptLanguage: "en-US,en",
	})

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

	page.Navigate(targetUrl)
	page.MustWaitLoad()
	logger.Info.Println("Waiting for idle..")
	page.MustWaitIdle()
	time.Sleep(time.Second)

	logger.Info.Println("Needs login?")
	if needsLogin(page) {
		logger.Info.Println("Yes, initiate login")
		initiateLogin(page)
		page.Reload()

		err = page.WaitIdle(5 * time.Second)
		if err != nil {
			return
		}

		if needsLogin(page) {
			return page, fmt.Errorf("login ran, but still need login")
		}
	}
	logger.Info.Println("No login")

	// Check cookie banner
	logger.Info.Println("Check for cookie banner")
	cookieBanner, _ := ElementImmediateRecursive(page, "#bnp_cookie_banner")
	if cookieBanner != nil {
		isVisible, _ := cookieBanner.Visible()
		if isVisible {
			fmt.Println("Cookie banner visible")
			cookieBannerRejectBtn, err := ElementImmediateRecursive(page, "#bnp_btn_reject")
			if err != nil {
				return nil, fmt.Errorf("cookieBanner reject-button error", err)
			}
			cookieBannerRejectBtn.MustEval("() => this.click()")
			fmt.Println("Cookie banner rejected")
		}
	}

	var inputElement *rod.Element
	err = RunFuncNTimes(10, time.Second*1, func() (err error) {
		logger.Info.Println("Get searchbox")
		inputElement, err = ElementImmediateRecursive(page, "#searchbox")
		return
	})
	if err != nil {
		page.MustScreenshot("/data/textbox_not_found.png")
		return nil, fmt.Errorf("textbox not found: %e", err)
	}

	// Select creative
	logger.Info.Println("Select creative")
	creativeBtn, _ := ElementImmediateRecursive(page, "button.tone-creative")
	creativeBtn.Eval("() => this.click()")

	time.Sleep(time.Duration(1) * time.Second)
	inputElement.Eval("() => this.removeAttribute('maxlength')")

	// Setup Copilot-settings
	logger.Info.Println("Setup copilot settings")
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
			break
		}

		log.Printf("Clicking %s\n", currAction.Selector) // Add log message
		elm.Click(proto.InputMouseButtonLeft, 1)
		time.Sleep(time.Duration(1) * time.Second)
	}

	newPage := GetFirstNotEquals(browser.MustPages(), page)
	if newPage == nil {
		//return fmt.Errorf("no login-page opened")
		newPage = page
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
	if page != newPage {
		newPage.Close()
	}

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
func handleWorkItem(work WorkItem) (err error) {
	page := work.Page
	defer close(work.OutputStream)

	inputElement, err := ElementImmediateRecursive(page, "#searchbox")
	if err != nil {
		return err
	}

	var running = true
	go page.EachEvent(func(e *proto.NetworkWebSocketCreated) {
		fmt.Println("Websocket created")
	}, func(e *proto.NetworkWebSocketFrameReceived) {
		fmt.Println("Websocket frame received:", len(e.Response.PayloadData))
		//if e.Response.Opcode != 1 { return }

		for _, payload := range strings.Split(e.Response.PayloadData, "\x1e") {
			if len(payload) == 0 {
				continue
			}

			work.OutputStream <- []byte(payload)
		}
	}, func(e *proto.NetworkWebSocketClosed) {
		fmt.Println("Websocket closed")
		running = false
	})()

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

	// Check for privacy banner
	privacyBannerBtn, _ := ElementImmediateRecursive(page, ".get-started-btn-wrapper button")
	if privacyBannerBtn != nil {
		fmt.Println("Privacy banner detected, accepting")
		privacyBannerBtn.MustEval("() => this.click()")
	}

	for running {
		select {
		case <-work.Context.Done():
			// Request was cancelled
			work.Page.Close() // TODO: Properly cancel BingChat
			return
		}
	}

	return nil
}

// processRequests processes work items from the work queue
func processRequests() {
	var curPage *rod.Page
	for {
		for curPage == nil {
			curPage, _ = GetNewReadyPage()
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

func GetManualConversation(page *rod.Page) (conversationId string, clientId string, signature string, err error) {
	obj, err := page.Evaluate(&rod.EvalOptions{
		JS: `(async () => {
			// Use the fetch API to send a GET request
			let response = await fetch("/turing/conversation/create?bundleVersion=1.1573.3");
			// Check if the response is ok
			if (!response.ok) {
				// Throw an error if the response is not ok
				throw new Error("An error occurred: ${response.status}");
			}

			let text = await response.text();
			let signature = response.headers.get("x-sydney-encryptedconversationsignature");

			return {
				"data": JSON.parse(text),
				"signature": signature
			};
		})`,
		UserGesture:  true,
		AwaitPromise: true,
		ByValue:      true})

	if err != nil {
		return
	}

	conversationId = obj.Value.Get("data.conversationId").String()
	clientId = obj.Value.Get("data.clientId").String()
	signature = obj.Value.Get("signature").String()

	result_value := obj.Value.Get("data.result.value").String()
	result_message := obj.Value.Get("data.result.message").String()

	if !strings.EqualFold(result_value, "Success") {
		err = fmt.Errorf("failed getting conversation: %s", result_message)
	}

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
	elms, err := page.ElementsByJS(&rod.EvalOptions{
		JS: `() => {
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
	`})

	if err != nil {
		return nil, err
	}

	if elms.Empty() {
		return nil, fmt.Errorf("no element on page")
	}

	return elms.First(), nil
}
