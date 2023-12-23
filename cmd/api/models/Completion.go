package models

type Choice struct {
	Index    int `json:"index"`
	Message  `json:"message"`
	Logprobs interface{} `json:"logprobs"`

	FinishReason string `json:"finish_reason"`
}
type Completion struct {
	ID      string `json:"id"`
	Object  string `json:"chat.completion"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Usage   struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Choices []Choice `json:"choices"`
}
