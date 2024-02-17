package reng

import (
	"context"
	"errors"
	"io"
	"os"
	//"time"

	"github.com/gorilla/websocket"
)

type ReadResult struct {
	messageType int
	data        []byte
	err         error
}

func readWssMessages(ctx context.Context, conn *websocket.Conn) <-chan ReadResult {
	ch := make(chan ReadResult)

	var messageType int
	var reader io.Reader
	var err error

	// Reserve space for messages
	//data := make([]byte, 32*1024) // 32KB

	go func() {
		defer close(ch) // close the channel when the function returns
		for {
			select {
			case <-ctx.Done(): // check if the context is cancelled
				return
			default:
				// set a deadline for reading from the conn
				//conn.SetReadDeadline(time.Now().Add(time.Millisecond * 100))

				// read from the conn
				messageType, reader, err = conn.NextReader()
				if err != nil {
					ch <- ReadResult{
						messageType,
						nil,
						err,
					}
					return
				}
				//n, err := reader.Read(data)
				data, err := io.ReadAll(reader)

				if err != nil && errors.Is(err, os.ErrDeadlineExceeded) {
					// if the error is a timeout, ignore
					continue
				}

				// Send result to channel
				ch <- ReadResult{
					messageType,
					//data[:n],
					data,
					err,
				}

				if err != nil {
					// If error is not timeout, kill
					return
				}
			}
		}
	}()

	return ch
}
