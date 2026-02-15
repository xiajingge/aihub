package httpclient

import (
	"bufio"
	"context"
	"errors"
	"io"
	"strings"
	"sync"
)

// StreamDecoder defines the interface for decoding streaming responses.
type StreamDecoder = Stream[*StreamEvent]

// StreamDecoderFactory is a function that creates a StreamDecoder from a ReadCloser.
type StreamDecoderFactory func(ctx context.Context, rc io.ReadCloser) StreamDecoder

// decoderRegistry holds registered stream decoders.
type decoderRegistry struct {
	mu       sync.RWMutex
	decoders map[string]StreamDecoderFactory
}

// globalRegistry is the global decoder registry.
var globalRegistry = &decoderRegistry{
	decoders: make(map[string]StreamDecoderFactory),
}

// RegisterDecoder registers a stream decoder for a specific content type.
func RegisterDecoder(contentType string, factory StreamDecoderFactory) {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()

	globalRegistry.decoders[contentType] = factory
}

// GetDecoder returns a decoder factory for the given content type.
func GetDecoder(contentType string) (StreamDecoderFactory, bool) {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	factory, exists := globalRegistry.decoders[contentType]

	return factory, exists
}

// NewDefaultSSEDecoder creates a new default SSE decoder.
func NewDefaultSSEDecoder(ctx context.Context, rc io.ReadCloser) StreamDecoder {
	return &defaultSSEDecoder{
		ctx:    ctx,
		rc:     rc,
		reader: bufio.NewReader(rc),
	}
}

// Ensure defaultSSEDecoder implements StreamDecoder.
var _ StreamDecoder = (*defaultSSEDecoder)(nil)

// defaultSSEDecoder implements streams.Stream for Server-Sent Events with a raw body reader.
//
//nolint:containedctx // Checked.
type defaultSSEDecoder struct {
	ctx         context.Context
	rc          io.ReadCloser
	reader      *bufio.Reader
	current     *StreamEvent
	err         error
	lastEventID string
	eof         bool
	closed      bool
}

// Next advances to the next event in the stream.
func (s *defaultSSEDecoder) Next() bool {
	if s.err != nil || s.eof {
		return false
	}

	// Check context cancellation
	select {
	case <-s.ctx.Done():
		s.err = s.ctx.Err()
		_ = s.Close()

		return false
	default:
	}

	eventType := ""
	dataLines := make([]string, 0, 4)
	nextLastEventID := s.lastEventID
	lastEventIDUpdated := false

	for {
		// 每次读取一行
		rawLine, readErr := s.reader.ReadString('\n')
		if readErr != nil && !errors.Is(readErr, io.EOF) {
			s.err = readErr
			_ = s.Close()

			return false
		}

		line := strings.TrimRight(rawLine, "\r\n")

		// 空行表示一个 SSE event 结束。
		if line == "" {
			if lastEventIDUpdated {
				s.lastEventID = nextLastEventID
			}

			if len(dataLines) > 0 || eventType != "" {
				s.current = &StreamEvent{
					LastEventID: s.lastEventID,
					Type:        eventType,
					Data:        []byte(strings.Join(dataLines, "\n")),
				}

				return true
			}

			eventType = ""
			dataLines = dataLines[:0]
			nextLastEventID = s.lastEventID
			lastEventIDUpdated = false
		} else {
			field, value, ok := parseSSEField(line)
			if ok {
				switch field {
				case "data":
					dataLines = append(dataLines, value)
				case "event":
					eventType = value
				case "id":
					if !strings.ContainsRune(value, '\x00') {
						nextLastEventID = value
						lastEventIDUpdated = true
					}
				case "retry":
					// retry 字段由上游重连逻辑处理，这里仅做解析兼容。
				default:
				}
			}
		}

		if errors.Is(readErr, io.EOF) {
			if lastEventIDUpdated {
				s.lastEventID = nextLastEventID
			}

			if len(dataLines) > 0 || eventType != "" {
				s.current = &StreamEvent{
					LastEventID: s.lastEventID,
					Type:        eventType,
					Data:        []byte(strings.Join(dataLines, "\n")),
				}
				s.eof = true
				_ = s.Close()

				return true
			}

			s.eof = true
			_ = s.Close()

			return false
		}
	}
}

// Current returns the current event data.
func (s *defaultSSEDecoder) Current() *StreamEvent {
	return s.current
}

// Err returns any error that occurred during streaming.
func (s *defaultSSEDecoder) Err() error {
	return s.err
}

// Close closes the stream and releases resources.
func (s *defaultSSEDecoder) Close() error {
	if s.closed {
		return nil
	}

	s.closed = true
	if s.rc != nil {
		return s.rc.Close()
	}

	return nil
}

func parseSSEField(line string) (field, value string, ok bool) {
	if strings.HasPrefix(line, ":") {
		return "", "", false
	}

	idx := strings.IndexByte(line, ':')
	if idx < 0 {
		return line, "", true
	}

	field = line[:idx]
	value = line[idx+1:]
	if len(value) > 0 && value[0] == ' ' {
		value = value[1:]
	}

	return field, value, true
}

// init registers the default SSE decoder.
func init() {
	RegisterDecoder("text/event-stream", NewDefaultSSEDecoder)
	RegisterDecoder("text/event-stream; charset=utf-8", NewDefaultSSEDecoder)
}
