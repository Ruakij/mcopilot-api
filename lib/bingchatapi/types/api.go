package types

import "context"

type Message struct {
	Role    string
	Content string
}

type WorkItem struct {
	Messages     []Message
	Context      context.Context
	OutputStream chan<- []byte
	Model        string
}

type Api interface {
	ProcessRequest(workItem *WorkItem)
	Init()
}
