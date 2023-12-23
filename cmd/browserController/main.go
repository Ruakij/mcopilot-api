package browsercontroller

import (
	"context"
	"fmt"
	"log"
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
	Email string
	Password string
	TotpSecret string
}

var loginData *LoginData
func Setup(setupLoginData *LoginData) (err error) {
	loginData = setupLoginData

	err = setupRod()
	if err != nil {
		return
	}

	go processRequests()

	return
}

type WorkItem struct {
	context      context.Context
	input        string
	outputStream chan<- BingChatResponse
}

var workQueue chan WorkItem = make(chan WorkItem)

// Place request into queue, blocks until work is processing
func ProcessChatRequest(context context.Context, text string, streamChan chan<- BingChatResponse) {
	work := WorkItem{
		context:      context,
		input:        text,
		outputStream: streamChan,
	}

	workQueue <- work
}

var browser *rod.Browser

func setupRod() (err error) {
	// Connect to the WebDriver instance running locally.

	launcher, err := launcher.New().
		//Headless(false).
		Launch()
	if err != nil {
		return err
	}
	browser = rod.New().ControlURL(launcher)
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

// handleWorkItem handles a single work item by sending the input to the page and receiving the output
func handleWorkItem(page *rod.Page, work WorkItem) error {
	defer page.Close()

	inputElement, err := ElementImmediateRecursive(page, "#searchbox")
	if err != nil {
		return err
	}

	fmt.Println("Ready!")
	fmt.Printf("Got request %s\n", work.input)

	waitChan := make(chan string)

	go page.EachEvent(func(e *proto.NetworkWebSocketCreated) {
		fmt.Println("created", e.URL)
		waitChan <- ""
	}, func(e *proto.NetworkWebSocketFrameReceived) {
		//if e.Response.Opcode != 1 { return }

		for _, payload := range strings.Split(e.Response.PayloadData, "\x1e") {
			if len(payload) == 0 {
				continue
			}

			fmt.Println("\n", payload)

			var bingChatResponse BingChatResponse
			err := json.Unmarshal([]byte(payload), &bingChatResponse)
			if err != nil {
				fmt.Println(err)
				continue
			}

			switch bingChatResponse.Type {
			case 1:
				if len(bingChatResponse.Arguments) > 0 && len(bingChatResponse.Arguments[len(bingChatResponse.Arguments)-1].Messages) > 0 {
					work.outputStream <- bingChatResponse
				}
			case 2:
				var bingChatResponseSummary BingChatResponseSummary
				json.Unmarshal([]byte(payload), &bingChatResponseSummary)
				if err != nil {
					continue
				}

				// Enter summary-data to stream
				work.outputStream <- BingChatResponse{
					Type:      bingChatResponseSummary.Type,
					Arguments: []BingChatResponseArgument{bingChatResponseSummary.Item},
				}

				// Tell others stream is over
				waitChan <- ""
			}
		}
	}, func(e *proto.NetworkWebSocketClosed) {
		waitChan <- ""
	})()

	// Type in
	inputElement.Eval("(input) => this.value = input", work.input)
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

	time.Sleep(100 * time.Millisecond)

	sendBtn, err := ElementImmediateRecursive(page, ".submit button")
	if err != nil {
		return err
	}
	sendBtn.Eval("() => this.click()")

	// Wait max 5s for stream to start
	streamStarted := false
	select {
	case <-waitChan:
		streamStarted = true
		fmt.Println("Stream started")
	case <-time.After(time.Second * 5):
		if !streamStarted {
			fmt.Println("Stream not started after 5s")
			close(work.outputStream)
			return nil
		}
	case <-work.context.Done():
		// Request was cancelled
		close(work.outputStream)
		return nil
	}

	select {
	case <-waitChan:
		close(work.outputStream)
		return nil
	case <-work.context.Done():
		close(work.outputStream)
		return nil
	}
}

// processRequests processes work items from the work queue
func processRequests() {
	for {
		page, err := getNewReadyPage()
		if err != nil {
			time.Sleep(30 * time.Second)
			continue
		}

		var work WorkItem = <-workQueue
		err = handleWorkItem(page, work)
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
