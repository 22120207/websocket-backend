package cmdrunner

// StreamSender defines the interface for anything that can send a byte slice message.
// The websocket.Client will implicitly implement this interface.
type StreamSender interface {
	Send(message []byte)
	UpdateState(newState bool)
}
