package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/xiajignge/aihub/internal/domain"
	"github.com/xiajignge/aihub/internal/domain/transformer"
	"github.com/xiajignge/aihub/pkg/httpclient"
)

// PlatformType represents the platform type for OpenAI API.
type PlatformType string

const (
	PlatformOpenAI PlatformType = "openai" // Standard OpenAI API
	PlatformAzure  PlatformType = "azure"  // Azure OpenAI
)

const DefaultAzureAPIVersion = "2025-04-01-preview"

// Config holds all configuration for the OpenAI outbound transformer.
type Config struct {
	// Platform configuration
	Type PlatformType `json:"type"`

	// API configuration
	BaseURL string `json:"base_url,omitempty"` // Custom base URL (optional)
	APIKey  string `json:"api_key,omitempty"`  // API key

	// Azure-specific configuration
	APIVersion string `json:"api_version,omitempty"` // Azure API version (required for Azure)
}

// OutboundTransformer implements transformer.Outbound for OpenAI format.
type OutboundTransformer struct {
	config *Config
}

// NewOutboundTransformer creates a new OpenAI OutboundTransformer with legacy parameters.
// Deprecated: Use NewOutboundTransformerWithConfig instead.
func NewOutboundTransformer(baseURL, apiKey string) (transformer.Outbound, error) {
	config := &Config{
		Type:    PlatformOpenAI,
		BaseURL: baseURL,
		APIKey:  apiKey,
	}

	err := validateConfig(config)
	if err != nil {
		return nil, fmt.Errorf("invalid OpenAI transformer configuration: %w", err)
	}

	return NewOutboundTransformerWithConfig(config)
}

// NewOutboundTransformerWithConfig creates a new OpenAI OutboundTransformer with unified configuration.
func NewOutboundTransformerWithConfig(config *Config) (transformer.Outbound, error) {
	err := validateConfig(config)
	if err != nil {
		return nil, fmt.Errorf("invalid OpenAI transformer configuration: %w", err)
	}

	return &OutboundTransformer{
		config: config,
	}, nil
}

// validateConfig validates the configuration for the given platform.
func validateConfig(config *Config) error {
	if config == nil {
		return errors.New("config cannot be nil")
	}

	// Standard OpenAI validation
	if config.APIKey == "" {
		return errors.New("API key is required")
	}

	if config.BaseURL == "" {
		return errors.New("base URL is required")
	}

	switch config.Type {
	case PlatformOpenAI:
		return nil
	case PlatformAzure:
		if config.APIVersion == "" {
			return fmt.Errorf("API version is required for Azure platform")
		}
	default:
		return fmt.Errorf("unsupported platform type: %v", config.Type)
	}

	return nil
}

// APIFormat returns the API format of the transformer.
func (t *OutboundTransformer) APIFormat() domain.APIFormat {
	return domain.APIFormatOpenAIChatCompletion
}

// TransformRequest transforms ChatCompletionRequest to Request.
func (t *OutboundTransformer) TransformRequest(
	ctx context.Context,
	chatReq *domain.Request,
) (*httpclient.Request, error) {
	if chatReq == nil {
		return nil, fmt.Errorf("chat completion request is nil")
	}

	// Validate required fields
	if chatReq.Model == "" {
		return nil, fmt.Errorf("model is required")
	}

	if len(chatReq.Messages) == 0 {
		return nil, fmt.Errorf("messages are required")
	}

	body, err := json.Marshal(chatReq)
	if err != nil {
		return nil, fmt.Errorf("failed to transform request: %w", err)
	}

	// Prepare headers
	headers := make(http.Header)
	headers.Set("Content-Type", "application/json")
	headers.Set("Accept", "application/json")

	var auth *httpclient.AuthConfig

	//nolint:exhaustive // Chcked.
	switch t.config.Type {
	case PlatformAzure:
		auth = &httpclient.AuthConfig{
			Type:      "api_key",
			APIKey:    t.config.APIKey,
			HeaderKey: "Api-Key",
		}
	default:
		auth = &httpclient.AuthConfig{
			Type:   "bearer",
			APIKey: t.config.APIKey,
		}
	}

	// Build platform-specific URL
	url, err := t.buildPlatformURL(chatReq)
	if err != nil {
		return nil, fmt.Errorf("failed to build platform URL: %w", err)
	}

	return &httpclient.Request{
		Method:  http.MethodPost,
		URL:     url,
		Headers: headers,
		Body:    body,
		Auth:    auth,
	}, nil
}

// TransformResponse transforms Response to ChatCompletionResponse.
func (t *OutboundTransformer) TransformResponse(
	ctx context.Context,
	httpResp *httpclient.Response,
) (*domain.Response, error) {
	if httpResp == nil {
		return nil, fmt.Errorf("http response is nil")
	}

	// Check for HTTP error status codes
	if httpResp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP error %d", httpResp.StatusCode)
	}

	// Check for empty response body
	if len(httpResp.Body) == 0 {
		return nil, fmt.Errorf("response body is empty")
	}

	var chatResp domain.Response

	err := json.Unmarshal(httpResp.Body, &chatResp)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal chat completion response: %w", err)
	}

	return &chatResp, nil
}

func (t *OutboundTransformer) TransformStream(ctx context.Context, stream httpclient.Stream[*httpclient.StreamEvent]) (httpclient.Stream[*domain.Response], error) {
	if stream == nil {
		return nil, fmt.Errorf("%w: stream is nil", transformer.ErrInvalidRequest)
	}

	return &openAIOutboundStream{
		ctx:         ctx,
		src:         stream,
		transformer: t,
	}, nil
}

func (t *OutboundTransformer) TransformStreamChunk(
	ctx context.Context,
	event *httpclient.StreamEvent,
) (*domain.Response, error) {
	if event == nil {
		return nil, fmt.Errorf("stream event is nil")
	}

	payload := bytes.TrimSpace(event.Data)
	if len(payload) == 0 {
		return nil, fmt.Errorf("stream event data is empty")
	}

	if bytes.Equal(payload, []byte("[DONE]")) {
		return domain.DoneResponse, nil
	}

	ep := gjson.GetBytes(payload, "error")
	if ep.Exists() {
		detail := domain.ErrorDetail{
			Code:      gjson.GetBytes(payload, "error.code").String(),
			Message:   gjson.GetBytes(payload, "error.message").String(),
			Type:      gjson.GetBytes(payload, "error.type").String(),
			Param:     gjson.GetBytes(payload, "error.param").String(),
			RequestID: gjson.GetBytes(payload, "error.request_id").String(),
		}
		if detail.Message == "" {
			detail.Message = ep.String()
		}
		if detail.Type == "" {
			detail.Type = "api_error"
		}

		return nil, &domain.ResponseError{
			Detail: detail,
		}
	}

	// Create a synthetic HTTP response for compatibility with existing logic
	httpResp := &httpclient.Response{
		Body: payload,
	}

	return t.TransformResponse(ctx, httpResp)
}

// buildPlatformURL constructs the appropriate URL based on the platform.
func (t *OutboundTransformer) buildPlatformURL(chatReq *domain.Request) (string, error) {
	baseURL := strings.TrimSuffix(t.config.BaseURL, "/")

	//nolint:exhaustive // Chcked.
	switch t.config.Type {
	case PlatformAzure:
		// Build the Azure OpenAI URL
		azureURL := fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=%s",
			baseURL, chatReq.Model, t.config.APIVersion)

		return azureURL, nil
	default:
		// Standard OpenAI API
		return baseURL + "/chat/completions", nil
	}
}

// SetAPIKey updates the API key.
func (t *OutboundTransformer) SetAPIKey(apiKey string) {
	t.config.APIKey = apiKey

	// Validate configuration after updating API key
	err := validateConfig(t.config)
	if err != nil {
		panic(fmt.Sprintf("invalid OpenAI transformer configuration after setting API key: %v", err))
	}
}

// SetBaseURL updates the base URL.
func (t *OutboundTransformer) SetBaseURL(baseURL string) {
	t.config.BaseURL = baseURL

	// Validate configuration after updating base URL
	err := validateConfig(t.config)
	if err != nil {
		panic(fmt.Sprintf("invalid OpenAI transformer configuration after setting base URL: %v", err))
	}
}

// SetConfig updates the entire configuration.
func (t *OutboundTransformer) SetConfig(config *Config) {
	// Validate configuration before setting
	err := validateConfig(config)
	if err != nil {
		panic(fmt.Sprintf("invalid OpenAI transformer configuration: %v", err))
	}

	t.config = config
}

// ConfigureForAzure configures the transformer for Azure OpenAI.
func (t *OutboundTransformer) ConfigureForAzure(resourceName, apiVersion, apiKey string) error {
	// Create new Azure configuration
	newConfig := &Config{
		Type:       PlatformAzure,
		APIVersion: apiVersion,
		APIKey:     apiKey,
	}

	// Set base URL only if resource name is provided
	if resourceName != "" {
		newConfig.BaseURL = fmt.Sprintf("https://%s.openai.azure.com", resourceName)
	}

	// Validate the new configuration
	err := validateConfig(newConfig)
	if err != nil {
		return fmt.Errorf("invalid Azure configuration: %w", err)
	}

	// Apply the validated configuration
	t.config = newConfig

	return nil
}

// GetConfig returns the current configuration.
func (t *OutboundTransformer) GetConfig() *Config {
	return t.config
}

func (t *OutboundTransformer) AggregateStreamChunks(
	ctx context.Context,
	chunks []*httpclient.StreamEvent,
) ([]byte, domain.ResponseMeta, error) {
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

		resp, err := t.TransformStreamChunk(ctx, chunk)
		if err != nil {
			return nil, domain.ResponseMeta{}, fmt.Errorf("failed to transform stream chunk %d: %w", idx, err)
		}

		if resp == nil || resp.Object == domain.DoneResponse.Object {
			continue
		}

		hasValidChunk = true
		mergeResponseHeader(&merged, resp)
		if resp.Usage != nil {
			merged.Usage = resp.Usage
		}
		if resp.Error != nil {
			merged.Error = resp.Error
		}

		for _, c := range resp.Choices {
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

type openAIOutboundStream struct {
	ctx         context.Context
	src         httpclient.Stream[*httpclient.StreamEvent]
	transformer *OutboundTransformer
	current     *domain.Response
	err         error
	closed      bool
	done        bool
}

func (s *openAIOutboundStream) Next() bool {
	if s.err != nil || s.done {
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
			}
			_ = s.Close()
			return false
		}

		event := s.src.Current()
		if event == nil {
			continue
		}

		resp, err := s.transformer.TransformStreamChunk(s.ctx, event)
		if err != nil {
			s.err = err
			_ = s.Close()
			return false
		}

		s.current = resp
		if resp != nil && resp.Object == domain.DoneResponse.Object {
			s.done = true
			_ = s.Close()
		}

		return true
	}
}

func (s *openAIOutboundStream) Current() *domain.Response {
	return s.current
}

func (s *openAIOutboundStream) Err() error {
	return s.err
}

func (s *openAIOutboundStream) Close() error {
	if s.closed {
		return nil
	}
	s.closed = true
	if s.src != nil {
		return s.src.Close()
	}
	return nil
}

// TransformError transforms HTTP error response to unified error response.
func (t *OutboundTransformer) TransformError(ctx context.Context, rawErr *httpclient.Error) *domain.ResponseError {
	if rawErr == nil {
		return &domain.ResponseError{
			StatusCode: http.StatusInternalServerError,
			Detail: domain.ErrorDetail{
				Message: http.StatusText(http.StatusInternalServerError),
				Type:    "api_error",
			},
		}
	}

	// Try to parse as OpenAI error format first
	var openaiError struct {
		Error domain.ErrorDetail `json:"error"`
	}

	err := json.Unmarshal(rawErr.Body, &openaiError)
	if err == nil && openaiError.Error.Message != "" {
		return &domain.ResponseError{
			StatusCode: rawErr.StatusCode,
			Detail:     openaiError.Error,
		}
	}

	// If JSON parsing fails, return the JSON error message
	return &domain.ResponseError{
		StatusCode: rawErr.StatusCode,
		Detail: domain.ErrorDetail{
			Message: http.StatusText(http.StatusInternalServerError),
			Type:    "api_error",
		},
	}
}
