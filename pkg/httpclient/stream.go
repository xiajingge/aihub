package httpclient

// Stream represents a generic stream interface
// The caller should check the Err() method to ensure there's no error.
type Stream[T any] interface {
	// Next indicate if there's a next item.
	Next() bool
	// Current returns the current event
	Current() T
	// Err returns any error that occurred
	Err() error
	// Close closes the stream
	Close() error
}

type StreamEvent struct {
	LastEventID string `json:"last_event_id,omitempty"`
	Type        string `json:"type"`
	Data        []byte `json:"data"`
}
