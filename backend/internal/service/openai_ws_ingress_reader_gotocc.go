package service

import (
	"context"

	coderws "github.com/coder/websocket"
)

// OpenAIWSIngressConn is the client transport shared by admission and forwarding.
type OpenAIWSIngressConn interface {
	Read(context.Context) (coderws.MessageType, []byte, error)
	Write(context.Context, coderws.MessageType, []byte) error
	Ping(context.Context) error
	Close(coderws.StatusCode, string) error
	CloseNow() error
}

type openAIWSIngressReader struct {
	*coderws.Conn
	ctx      context.Context
	cancel   context.CancelFunc
	messages chan openAIWSClientReadResult
	done     chan struct{}
	readErr  error
}

// NewOpenAIWSIngressReader gives the accepted connection one reader for its
// entire lifetime. Admission waits therefore observe a client disconnect even
// while no turn is consuming messages. One prefetched frame retains backpressure;
// all frames still pass through the existing per-turn validation and audit.
func NewOpenAIWSIngressReader(parent context.Context, conn *coderws.Conn) (context.Context, OpenAIWSIngressConn) {
	ctx, cancel := context.WithCancel(parent)
	r := &openAIWSIngressReader{
		Conn: conn, ctx: ctx, cancel: cancel,
		messages: make(chan openAIWSClientReadResult, 1),
		done:     make(chan struct{}),
	}
	go r.readLoop()
	return ctx, r
}

func (r *openAIWSIngressReader) readLoop() {
	defer close(r.done)
	for {
		messageType, payload, err := r.Conn.Read(context.Background())
		if err != nil {
			r.readErr = err
			r.cancel()
			return
		}
		select {
		case r.messages <- openAIWSClientReadResult{messageType: messageType, payload: payload}:
		case <-r.ctx.Done():
			return
		}
	}
}

func (r *openAIWSIngressReader) Read(ctx context.Context) (coderws.MessageType, []byte, error) {
	select {
	case message := <-r.messages:
		return message.messageType, message.payload, nil
	case <-r.done:
		if r.readErr != nil {
			return 0, nil, r.readErr
		}
		return 0, nil, r.ctx.Err()
	case <-ctx.Done():
		return 0, nil, ctx.Err()
	}
}

func (r *openAIWSIngressReader) CloseNow() error {
	r.cancel()
	err := r.Conn.CloseNow()
	<-r.done
	return err
}
