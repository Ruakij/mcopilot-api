package models

type ChatRequest struct {
	Model            string    `json:"model"`
	Stream           bool      `json:"stream"`
	Messages         []Message `json:"messages"`
	FrequencyPenalty float32   `json:"frequency_penalty"`
	PresencePenalty  float32   `json:"presence_penalty"`
	Temperature      float32   `json:"temperature"`
	TopP             float32   `json:"top_p"`
}
