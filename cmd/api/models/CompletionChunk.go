package models

type CompletionChunkChoice struct {
	Index        uint        `json:"index"`
	Delta        Message       `json:"delta"`
	Logprobs     interface{} `json:"logprobs"`
	FinishReason string      `json:"finish_reason,omitempty"`
}
type CompletionChunk struct {
	ID                string                  `json:"id"`
	Object            string                  `json:"object"` // chat.completion.chunk
	Created           int64                   `json:"created"`
	Model             string                  `json:"model"`
	SystemFingerprint string                  `json:"system_fingerprint,omitempty"`
	Choices           []CompletionChunkChoice `json:"choices"`
}
