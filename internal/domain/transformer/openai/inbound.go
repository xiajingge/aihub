package openai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/xiajignge/aihub/internal/domain"
	"github.com/xiajignge/aihub/internal/domain/transformer"
	"github.com/xiajignge/aihub/pkg/httpclient"
)

type InboundTransformer struct{}

func (t *InboundTransformer) APIFormat() domain.APIFormat {
	return domain.APIFormatOpenAIChatCompletion
}

func (t *InboundTransformer) TransformRequest(ctx context.Context, httpReq *httpclient.Request) (*domain.Request, error) {
	if httpReq == nil {
		return nil, fmt.Errorf("%w: http request is nil", transformer.ErrInvalidRequest)
	}

	if len(httpReq.Body) == 0 {
		return nil, fmt.Errorf("%w: request body is empty", transformer.ErrInvalidRequest)
	}

	// Check content type
	contentType := httpReq.Headers.Get("Content-Type")
	if contentType == "" {
		contentType = httpReq.Headers.Get("Content-Type")
	}

	if !strings.Contains(strings.ToLower(contentType), "application/json") {
		return nil, fmt.Errorf("%w: unsupported content type: %s", transformer.ErrInvalidRequest, contentType)
	}

	var chatReq domain.Request

	err := json.Unmarshal(httpReq.Body, &chatReq)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to decode openai request: %w", transformer.ErrInvalidRequest, err)
	}

	// Validate required fields
	if chatReq.Model == "" {
		return nil, fmt.Errorf("%w: model is required", transformer.ErrInvalidRequest)
	}

	if len(chatReq.Messages) == 0 {
		return nil, fmt.Errorf("%w: messages are required", transformer.ErrInvalidRequest)
	}

	return &chatReq, nil
}

func (t *InboundTransformer) TransformResponse(ctx context.Context, chatResp *domain.Response) (*httpclient.Response, error) {
	if chatResp == nil {
		return nil, fmt.Errorf("chat completion response is nil")
	}

	body, err := json.Marshal(chatResp)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal chat completion response: %w", err)
	}

	// Create generic response
	return &httpclient.Response{
		StatusCode: http.StatusOK,
		Body:       body,
		Headers: http.Header{
			"Content-Type":  []string{"application/json"},
			"Cache-Control": []string{"no-cache"},
		},
	}, nil
}

func (t *InboundTransformer) TransformStream(ctx context.Context, stream httpclient.Stream[*domain.Response]) (httpclient.Stream[*httpclient.StreamEvent], error) {
	if stream == nil {
		return nil, fmt.Errorf("%w: stream is nil", transformer.ErrInvalidRequest)
	}

	return &openAIInboundStream{
		ctx: ctx,
		src: stream,
	}, nil
}

func (t *InboundTransformer) TransformError(ctx context.Context, err error) *httpclient.Error {
	if err == nil {
		return nil
	}

	var httpErr *httpclient.Error
	if ok := errors.As(err, &httpErr); ok && httpErr != nil {
		return httpErr
	}

	var respErr *domain.ResponseError
	if ok := errors.As(err, &respErr); ok && respErr != nil {
		statusCode := respErr.StatusCode
		if statusCode <= 0 {
			statusCode = http.StatusBadGateway
		}

		body, marshalErr := json.Marshal(respErr)
		if marshalErr != nil {
			body = []byte(respErr.Error())
		}

		return &httpclient.Error{
			Method:     http.MethodPost,
			URL:        "/v1/chat/completions",
			StatusCode: statusCode,
			Status:     http.StatusText(statusCode),
			Body:       body,
		}
	}

	return &httpclient.Error{
		Method:     http.MethodPost,
		URL:        "/v1/chat/completions",
		StatusCode: http.StatusInternalServerError,
		Status:     http.StatusText(http.StatusInternalServerError),
		Body:       []byte(err.Error()),
	}
}

func (t *InboundTransformer) AggregateStreamChunks(ctx context.Context, chunks []*httpclient.StreamEvent) ([]byte, domain.ResponseMeta, error) {
	var (
		merged         domain.Response
		meta           domain.ResponseMeta
		hasValidChunk  bool
		choicesByIndex = map[int]*domain.Choice{}
		choiceOrder    []int
	)

	for idx, chunk := range chunks {
		if chunk == nil || len(chunk.Data) == 0 {
			continue
		}

		payload := strings.TrimSpace(string(chunk.Data))
		if payload == "" || payload == "[DONE]" {
			continue
		}

		var cur domain.Response
		if err := json.Unmarshal(chunk.Data, &cur); err != nil {
			return nil, domain.ResponseMeta{}, fmt.Errorf("failed to unmarshal stream chunk %d: %w", idx, err)
		}

		hasValidChunk = true
		mergeResponseHeader(&merged, &cur)
		if cur.Usage != nil {
			merged.Usage = cur.Usage
		}
		if cur.Error != nil {
			merged.Error = cur.Error
		}

		for _, c := range cur.Choices {
			acc, exists := choicesByIndex[c.Index]
			if !exists {
				acc = &domain.Choice{
					Index: c.Index,
				}
				choicesByIndex[c.Index] = acc
				choiceOrder = append(choiceOrder, c.Index)
			}

			if c.Message != nil {
				acc.Message = cloneMessage(c.Message)
			}
			if c.Delta != nil {
				if acc.Message == nil {
					acc.Message = &domain.Message{}
				}
				mergeMessageDelta(acc.Message, c.Delta)
			}
			if c.FinishReason != nil {
				acc.FinishReason = c.FinishReason
			}
			if c.Logprobs != nil {
				acc.Logprobs = c.Logprobs
			}
		}
	}

	if !hasValidChunk {
		return nil, domain.ResponseMeta{}, fmt.Errorf("no valid stream chunks")
	}

	sort.Ints(choiceOrder)
	merged.Choices = make([]domain.Choice, 0, len(choiceOrder))
	for _, index := range choiceOrder {
		merged.Choices = append(merged.Choices, *choicesByIndex[index])
	}

	if merged.Object == "" {
		merged.Object = "chat.completion"
	}

	body, err := json.Marshal(merged)
	if err != nil {
		return nil, domain.ResponseMeta{}, fmt.Errorf("failed to marshal aggregated response: %w", err)
	}

	meta = domain.ResponseMeta{
		ID:    merged.ID,
		Usage: merged.Usage,
	}

	return body, meta, nil
}

// NewInboundTransformer creates a new OpenAI InboundTransformer.
func NewInboundTransformer() *InboundTransformer {
	return &InboundTransformer{}
}

type openAIInboundStream struct {
	ctx        context.Context
	src        httpclient.Stream[*domain.Response]
	current    *httpclient.StreamEvent
	err        error
	closed     bool
	doneSent   bool
	sourceDone bool
}

// Next 将内部统一响应转换为 OpenAI SSE 事件。
func (s *openAIInboundStream) Next() bool {
	if s.err != nil {
		return false
	}

	if s.doneSent {
		return false
	}

	select {
	case <-s.ctx.Done():
		s.err = s.ctx.Err()
		_ = s.Close()
		return false
	default:
	}

	for {
		if !s.src.Next() {
			if err := s.src.Err(); err != nil {
				s.err = err
				_ = s.Close()
				return false
			}

			if s.sourceDone {
				return false
			}

			// 若上游流结束但未显式返回 [DONE]，这里兜底补一个结束事件。
			s.current = makeDoneEvent()
			s.doneSent = true
			s.sourceDone = true
			_ = s.Close()
			return true
		}

		resp := s.src.Current()
		if resp == nil {
			continue
		}

		if resp.Object == domain.DoneResponse.Object {
			s.current = makeDoneEvent()
			s.doneSent = true
			s.sourceDone = true
			_ = s.Close()
			return true
		}

		payload, err := json.Marshal(resp)
		if err != nil {
			s.err = fmt.Errorf("failed to marshal stream response: %w", err)
			_ = s.Close()
			return false
		}

		s.current = &httpclient.StreamEvent{
			LastEventID: resp.ID,
			Type:        "",
			Data:        payload,
		}
		return true
	}
}

func (s *openAIInboundStream) Current() *httpclient.StreamEvent {
	return s.current
}

func (s *openAIInboundStream) Err() error {
	return s.err
}

func (s *openAIInboundStream) Close() error {
	if s.closed {
		return nil
	}
	s.closed = true
	if s.src != nil {
		return s.src.Close()
	}
	return nil
}

func makeDoneEvent() *httpclient.StreamEvent {
	return &httpclient.StreamEvent{
		Type: "",
		Data: append([]byte(nil), domain.DoneStreamEvent.Data...),
	}
}

func mergeResponseHeader(dst, src *domain.Response) {
	if src.ID != "" {
		dst.ID = src.ID
	}
	if src.Object != "" && src.Object != domain.DoneResponse.Object {
		dst.Object = src.Object
	}
	if src.Created != 0 {
		dst.Created = src.Created
	}
	if src.Model != "" {
		dst.Model = src.Model
	}
	if src.SystemFingerprint != "" {
		dst.SystemFingerprint = src.SystemFingerprint
	}
	if src.ServiceTier != "" {
		dst.ServiceTier = src.ServiceTier
	}
}

func mergeMessageDelta(dst, delta *domain.Message) {
	if delta == nil {
		return
	}

	if delta.Role != "" {
		dst.Role = delta.Role
	}
	if delta.Name != nil {
		name := *delta.Name
		dst.Name = &name
	}
	if delta.ToolCallID != nil {
		toolCallID := *delta.ToolCallID
		dst.ToolCallID = &toolCallID
	}
	if delta.ToolCallIsError != nil {
		isError := *delta.ToolCallIsError
		dst.ToolCallIsError = &isError
	}
	if delta.Refusal != "" {
		dst.Refusal += delta.Refusal
	}
	if delta.ReasoningContent != nil {
		if dst.ReasoningContent == nil {
			dst.ReasoningContent = ptrTo("")
		}
		*dst.ReasoningContent += *delta.ReasoningContent
	}

	if delta.Content.Content != nil {
		if dst.Content.Content == nil {
			dst.Content.Content = ptrTo("")
		}
		*dst.Content.Content += *delta.Content.Content
	}

	if len(delta.Content.MultipleContent) > 0 {
		dst.Content.MultipleContent = append(dst.Content.MultipleContent, delta.Content.MultipleContent...)
	}

	if len(delta.ToolCalls) > 0 {
		mergeToolCalls(dst, delta.ToolCalls)
	}
}

func mergeToolCalls(dst *domain.Message, incoming []domain.ToolCall) {
	for _, inc := range incoming {
		pos := -1
		for i := range dst.ToolCalls {
			if dst.ToolCalls[i].Index == inc.Index {
				pos = i
				break
			}
		}

		if pos < 0 {
			newCall := inc
			dst.ToolCalls = append(dst.ToolCalls, newCall)
			continue
		}

		existing := &dst.ToolCalls[pos]
		if inc.ID != "" {
			existing.ID = inc.ID
		}
		if inc.Type != "" {
			existing.Type = inc.Type
		}
		if inc.Function.Name != "" {
			existing.Function.Name = inc.Function.Name
		}
		if inc.Function.Arguments != "" {
			existing.Function.Arguments += inc.Function.Arguments
		}
	}
}

func cloneMessage(src *domain.Message) *domain.Message {
	if src == nil {
		return nil
	}
	cloned := *src

	if src.Name != nil {
		name := *src.Name
		cloned.Name = &name
	}
	if src.ToolCallID != nil {
		toolCallID := *src.ToolCallID
		cloned.ToolCallID = &toolCallID
	}
	if src.ToolCallIsError != nil {
		isError := *src.ToolCallIsError
		cloned.ToolCallIsError = &isError
	}
	if src.ReasoningContent != nil {
		reasoning := *src.ReasoningContent
		cloned.ReasoningContent = &reasoning
	}
	if src.Content.Content != nil {
		content := *src.Content.Content
		cloned.Content.Content = &content
	}
	if len(src.Content.MultipleContent) > 0 {
		cloned.Content.MultipleContent = append([]domain.MessageContentPart(nil), src.Content.MultipleContent...)
	}
	if len(src.ToolCalls) > 0 {
		cloned.ToolCalls = append([]domain.ToolCall(nil), src.ToolCalls...)
	}

	return &cloned
}

func ptrTo[T any](v T) *T {
	return &v
}
