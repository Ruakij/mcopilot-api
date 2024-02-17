package models

type CompletionChunkChoice struct {
	Index        uint        `json:"index,omitempty"`
	Delta        Message     `json:"delta"`
	Logprobs     interface{} `json:"logprobs,omitempty"`
	FinishReason string      `json:"finish_reason,omitempty"`
}
type CompletionChunk struct {
	ID                string                  `json:"id,omitempty"`
	Object            string                  `json:"object,omitempty"` // chat.completion.chunk
	Created           int64                   `json:"created,omitempty"`
	Model             string                  `json:"model,omitempty"`
	SystemFingerprint string                  `json:"system_fingerprint,omitempty"`
	Choices           []CompletionChunkChoice `json:"choices"`
}
