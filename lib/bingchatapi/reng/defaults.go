package reng

import (
	"fmt"
	"time"
)

// Defaults is a struct that contains some default values for the chat request
type DefaultsSet struct {
	delimiter     string
	bundleVersion string
	headers       map[string]string
	cookies       map[string]string
}

var Defaults DefaultsSet = DefaultsSet{
	delimiter:     "\x1e",
	bundleVersion: "1.1418.9-suno",
	headers: map[string]string{
		"accept":                      "*/*",
		"accept-language":             "en-US,en",
		"user-agent":                  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.2210.91",
		"sec-ch-ua":                   `"Chromium";v="120", "Not A(Brand";v="24", "Microsoft Edge";v="120"`,
		"sec-ch-ua-arch":              `"x86"`,
		"sec-ch-ua-bitness":           `"64"`,
		"sec-ch-ua-full-version":      `"120.0.2210.91"`,
		"sec-ch-ua-full-version-list": `"Chromium";v="120.0.2210.91", "Not A(Brand";v="24.0.0.0", "Microsoft Edge";v="120.0.2210.91"`,
		"sec-ch-ua-mobile":            "?0",
		"sec-ch-ua-model":             `""`,
		"sec-ch-ua-platform":          `"Windows"`,
		"sec-ch-ua-platform-version":  `"15.0.0"`,
		"sec-fetch-dest":              "empty",
		"sec-fetch-mode":              "cors",
		"sec-fetch-site":              "same-origin",
	},
	cookies: map[string]string{
		"SRCHD":          "AF=NOFORM",
		"PPLState":       "1",
		"KievRPSSecAuth": "",
		"SUID":           "",
		"SRCHUSR":        "",
		"SRCHHPGUSR":     fmt.Sprintf("HV=%d", int(time.Now().Unix())),
	},
}
