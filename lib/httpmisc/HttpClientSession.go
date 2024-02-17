package httpmisc

import (
	"io"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

// Client is a custom type that embeds http.Client and has default headers, cookies, and timeout
type HttpClientSession struct {
	http.Client
	DefaultHeaders map[string]string
	DefaultCookies map[string]string
	DefaultTimeout time.Duration
}

// Creates a new Client with the given default values
func NewHttpClientSession(headers, cookies map[string]string, timeout time.Duration) *HttpClientSession {
	return &HttpClientSession{
		Client: http.Client{
			Timeout: timeout,
		},
		DefaultHeaders: headers,
		DefaultCookies: cookies,
		DefaultTimeout: timeout,
	}
}

// CreateRequest creates a new request in the context of an HttpClientSession with default Settings
func (c *HttpClientSession) NewRequest(method, url string, body io.Reader) (*http.Request, error) {
	// Create a new request
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	// Set headers from the map
	for key, value := range c.DefaultHeaders {
		req.Header.Set(key, value)
	}

	// Set cookies from the map
	for name, value := range c.DefaultCookies {
		req.AddCookie(&http.Cookie{Name: name, Value: value})
	}

	// Return the request
	return req, nil
}

func (c *HttpClientSession) WSConnect(url string, header http.Header) (conn *websocket.Conn, err error) {
	if header == nil {
		header = http.Header{}
	}

	// Create a header with the default headers and cookies
	for key, value := range c.DefaultHeaders {
		header.Set(key, value)
	}
	cookieStr := ""
	for name, value := range c.DefaultCookies {
		cookieStr += name + "=" + value + "; "
	}
	header.Set("Cookie", cookieStr)

	// Dial the WebSocket server
	conn, _, err = websocket.DefaultDialer.Dial(url, header)
	if err != nil {
		// Handle error
		return
	}

	return
}
