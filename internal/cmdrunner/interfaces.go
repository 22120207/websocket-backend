package cmdrunner

import "context"

type StreamSender interface {
	Send(message []byte)
	UpdateState(newState bool)
	SetCancelFunc(cancel context.CancelFunc)
}
