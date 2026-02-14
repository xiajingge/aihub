package domain

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/xiajignge/aihub/pkg/httpclient"
)

var (
	DoneStreamEvent = httpclient.StreamEvent{
		Data: []byte("[DONE]"),
	}

	DoneResponse = &Response{
		Object: "[DONE]",
	}
)

// Request is the unified LLM request model used by the gateway layer.
// It is primarily compatible with OpenAI ChatCompletion format,
// while being extended to support multi-provider routing (Anthropic, AI SDK, etc).
//
// Design Goals:
//   - Cross-provider compatibility
//   - Streaming support
//   - Tool/function calling support
//   - Reasoning model support
//   - Gateway-level control & observability
type Request struct {

	// ============================================================
	// 1️⃣ 核心必填字段 (LLM execution fundamentals)
	// ============================================================

	// Messages represents the conversation history sent to the model.
	// Must contain at least one message.
	Messages []Message `json:"messages" validator:"required,min=1"`

	// Model specifies the model ID used to generate the response.
	// e.g. "gpt-4o", "deepseek-reasoner", "claude-3-opus"
	Model string `json:"model" validator:"required"`

	// ============================================================
	// 2️⃣生成控制相关参数
	// (Controls randomness, creativity, output length)
	// ============================================================

	// Temperature controls randomness (0~2).
	// Lower = deterministic, Higher = creative.
	Temperature *float64 `json:"temperature,omitempty"`

	// TopP enables nucleus sampling as an alternative to temperature.
	// Recommended not to tune both Temperature and TopP simultaneously.
	TopP *float64 `json:"top_p,omitempty"`

	// FrequencyPenalty reduces repetition of tokens already generated.
	FrequencyPenalty *float64 `json:"frequency_penalty,omitempty"`

	// PresencePenalty increases likelihood of introducing new topics.
	PresencePenalty *float64 `json:"presence_penalty,omitempty"`

	// MaxCompletionTokens is the preferred way to limit generated tokens.
	MaxCompletionTokens *int64 `json:"max_completion_tokens,omitempty"`

	// Deprecated: use MaxCompletionTokens instead.
	MaxTokens *int64 `json:"max_tokens,omitempty"`

	// Seed attempts deterministic sampling when supported.
	// Not guaranteed across backend changes.
	Seed *int64 `json:"seed,omitempty"`

	// ============================================================
	// 3️⃣ 流式控制
	// ============================================================

	// Stream enables streaming (if true return chunk else SSE).
	Stream *bool `json:"stream,omitempty"`

	// StreamOptions provides additional control over stream behavior.
	// (current only include "IncludeUsage,if IncludeUsgae is true,it will add usage info to the last chunk")
	StreamOptions *StreamOptions `json:"stream_options,omitempty"`

	// ============================================================
	// 4️⃣ Tool / Function Calling
	// ============================================================

	// Tools defines callable functions exposed to the model.
	Tools []Tool `json:"tools,omitempty"`

	// ToolChoice controls whether/how a tool should be invoked.
	// Can be:
	//   - "auto"
	//   - "none"
	//   - specific named function
	ToolChoice *ToolChoice `json:"tool_choice,omitempty"`

	// ParallelToolCalls allows multiple tool calls in a single turn.
	ParallelToolCalls *bool `json:"parallel_tool_calls,omitzero"`

	// ============================================================
	// 5️⃣ Output & Formatting Control
	// ============================================================

	// ResponseFormat enforces structured output (e.g. JSON mode).
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`

	// Stop specifies stop sequences (string or string array).
	Stop *Stop `json:"stop,omitempty"`

	// ============================================================
	// 6️⃣ 日志功能（调试功能）
	// ============================================================

	// Logprobs enables token-level probability output.
	Logprobs *bool `json:"logprobs,omitempty"`

	// TopLogprobs returns top-k token alternatives per position.
	TopLogprobs *int64 `json:"top_logprobs,omitzero"`

	// LogitBias modifies likelihood of specific tokens.
	LogitBias map[string]int64 `json:"logit_bias,omitempty"`

	// ============================================================
	// 7️⃣ Reasoning Model Extensions
	// ============================================================

	// ReasoningEffort controls reasoning depth.
	// Supported values: "low", "medium", "high".
	ReasoningEffort string `json:"reasoning_effort,omitempty"`

	// ============================================================
	// 8️⃣ Caching, Safety & User Attribution
	// ============================================================

	// PromptCacheKey enables provider-side caching optimization.
	PromptCacheKey *bool `json:"prompt_cache_key,omitzero"`

	// SafetyIdentifier is a stable hashed user identifier.
	SafetyIdentifier *string `json:"safety_identifier,omitzero"`

	// Deprecated user identifier (legacy OpenAI field).
	User *string `json:"user,omitempty"`

	// ============================================================
	// 9️⃣ 成本与分层 Gateway-Level Service Control
	// ============================================================

	// ServiceTier controls request routing tier (e.g., free/standard/premium).
	ServiceTier *string `json:"service_tier,omitempty"`

	// Metadata stores up to 16 key-value pairs for analytics & tracing.
	Metadata map[string]string `json:"metadata,omitempty"`

	// ============================================================
	// 🔟 Internal Gateway Helper Fields (NOT sent to providers)
	// ============================================================

	// RawRequest holds the original incoming HTTP request.
	// Used for logging, replay, or protocol conversion.
	RawRequest *httpclient.Request `json:"-"`

	// RawAPIFormat indicates the original upstream protocol format.
	// e.g.:
	//   - openai/chat_completions
	//   - anthropic/messages
	//   - aisdk/text
	RawAPIFormat APIFormat `json:"-"`
}

type APIFormat string

const (
	APIFormatOpenAIChatCompletion APIFormat = "openai/chat_completions"
	APIFormatOpenAIResponse       APIFormat = "openai/response"
	APIFormatAnthropicMessage     APIFormat = "anthropic/messages"
	APIFormatAiSDKText            APIFormat = "aisdk/text"
	APIFormatAiSDKDataStream      APIFormat = "aisdk/datastream"
)

type ToolFunction struct {
	Name string `json:"name"`
}

// ToolChoice represents the tool choice parameter for function calling.
//
// Tool choice can be a string or a struct.
type ToolChoice struct {
	ToolChoice      *string          `json:"tool_choice,omitempty"`
	NamedToolChoice *NamedToolChoice `json:"named_tool_choice,omitempty"`
}

type NamedToolChoice struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

func (t ToolChoice) MarshalJSON() ([]byte, error) {
	if t.ToolChoice != nil {
		return json.Marshal(t.ToolChoice)
	}

	return json.Marshal(t.NamedToolChoice)
}

func (t *ToolChoice) UnmarshalJSON(data []byte) error {
	var str string

	err := json.Unmarshal(data, &str)
	if err == nil {
		t.ToolChoice = &str
		return nil
	}

	var named NamedToolChoice

	err = json.Unmarshal(data, &named)
	if err == nil {
		t.NamedToolChoice = &named
		return nil
	}

	return errors.New("invalid tool choice type")
}

type StreamOptions struct {
	// If set, an additional chunk will be streamed before the data: [DONE] message.
	// The usage field on this chunk shows the token usage statistics for the entire request,
	// and the choices field will always be an empty array.
	// All other chunks will also include a usage field, but with a null value.
	IncludeUsage bool `json:"include_usage,omitempty"`
}

type Stop struct {
	Stop         *string
	MultipleStop []string
}

func (s Stop) MarshalJSON() ([]byte, error) {
	if s.Stop != nil {
		return json.Marshal(s.Stop)
	}

	if len(s.MultipleStop) > 0 {
		return json.Marshal(s.MultipleStop)
	}

	return []byte("[]"), nil
}

func (s *Stop) UnmarshalJSON(data []byte) error {
	var str string

	err := json.Unmarshal(data, &str)
	if err == nil {
		s.Stop = &str
		return nil
	}

	var strs []string

	err = json.Unmarshal(data, &strs)
	if err == nil {
		s.MultipleStop = strs
		return nil
	}

	return errors.New("invalid stop type")
}

// Message represents a message in the conversation.
type Message struct {
	Role    string         `json:"role"`
	Content MessageContent `json:"content"` // string or []ContentPart
	Name    *string        `json:"name,omitempty"`

	// The refusal message generated by the model.
	Refusal string `json:"refusal,omitempty"`

	// For tool call response.

	// The index of the message that the tool call is associated with.
	// Is is a help field, will not be sent to the llm service.
	MessageIndex *int    `json:"-"`
	ToolCallID   *string `json:"tool_call_id,omitempty"`
	// This field is a help field, will not be sent to the llm service.
	ToolCallIsError *bool      `json:"-"`
	ToolCalls       []ToolCall `json:"tool_calls,omitempty"`

	// This property is used for the "reasoning" feature supported by deepseek-reasoner
	// the doc from deepseek:
	// - https://api-docs.deepseek.com/api/create-chat-completion#responses
	ReasoningContent *string `json:"reasoning_content,omitempty"`
}

type MessageContent struct {
	Content         *string              `json:"content,omitempty"`
	MultipleContent []MessageContentPart `json:"multiple_content,omitempty"`
}

func (c MessageContent) MarshalJSON() ([]byte, error) {
	if len(c.MultipleContent) > 0 {
		if len(c.MultipleContent) == 1 && c.MultipleContent[0].Type == "text" {
			return json.Marshal(c.MultipleContent[0].Text)
		}

		return json.Marshal(c.MultipleContent)
	}

	if c.Content != nil {
		return json.Marshal(c.Content)
	}

	return []byte(`""`), nil
}

func (c *MessageContent) UnmarshalJSON(data []byte) error {
	var str string

	err := json.Unmarshal(data, &str)
	if err == nil {
		c.Content = &str
		return nil
	}

	var parts []MessageContentPart

	err = json.Unmarshal(data, &parts)
	if err == nil {
		c.MultipleContent = parts
		return nil
	}

	return errors.New("invalid content type")
}

// MessageContentPart represents different types of content (text, image, etc.)
type MessageContentPart struct {
	// Type is the type of the content part.
	// e.g. "text", "image_url"
	Type string `json:"type"`
	// Text is the text content, required when type is "text"
	Text *string `json:"text,omitempty"`

	// ImageURL is the image URL content, required when type is "image_url"
	ImageURL *ImageURL `json:"image_url,omitempty"`

	// Audio is the audio content, required when type is "input_audio"
	Audio *Audio `json:"audio,omitempty"`
}

// ImageURL represents an image URL with optional detail level.
type ImageURL struct {
	// URL is the URL of the image.
	URL string `json:"url"`

	// Specifies the detail level of the image. Learn more in the
	// [Vision guide](https://platform.openai.com/docs/guides/vision#low-or-high-fidelity-image-understanding).
	//
	// Any of "auto", "low", "high".
	Detail string `json:"detail,omitempty"`
}

type Audio struct {
	// The format of the encoded audio data. Currently supports "wav" and "mp3".
	//
	// Any of "wav", "mp3".
	Format string `json:"format"`

	// Base64 encoded audio data.
	Data string `json:"data"`
}

// Tool represents a function tool.
type Tool struct {
	Type     string   `json:"type"`
	Function Function `json:"function"`
}

// FunctionRequest represents a function definition.
type Function struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters"`
}

// FunctionCall represents a function call (deprecated).
type FunctionCall struct {
	// The name of the function to call.
	Name string `json:"name"`

	// The arguments to call the function with, as generated by the model in JSON
	// format. Note that the model does not always generate valid JSON, and may
	// hallucinate parameters not defined by your function schema. Validate the
	// arguments in your code before calling your function.
	Arguments string `json:"arguments"`
}

// ToolCall represents a tool call in the response.
type ToolCall struct {
	ID string `json:"id,omitempty"`

	// The type of the tool. Currently, only `function` is supported.
	Type string `json:"type,omitempty"`

	Function FunctionCall `json:"function"`

	// The index of the tool call in the list of tool calls.
	Index int `json:"index,omitempty"`
}

// ResponseFormat specifies the format of the response.
type ResponseFormat struct {
	Type string `json:"type"`
	// TODO: Schema
}

// Response is the unified response model.
// To reduce the work of converting the response, we use the OpenAI response format.
// And other llm provider should convert the response to this format.
// NOTE: the OpenAI stream and non-stream response reuse same struct.
type Response struct {
	ID string `json:"id"`

	// A list of chat completion choices. Can be more than one if `n` is greater
	// than 1.
	Choices []Choice `json:"choices"`

	// Object is the type of the response.
	// e.g. "chat.completion", "chat.completion.chunk"
	Object string `json:"object"`

	// Created is the timestamp of when the response was created.
	Created int64 `json:"created"`

	// Model is the model used to generate the response.
	Model string `json:"model"`

	// An optional field that will only be present when you set stream_options: {"include_usage": true} in your request.
	// When present, it contains a null value except for the last chunk which contains the token usage statistics
	// for the entire request.
	Usage *Usage `json:"usage,omitempty"`

	// This fingerprint represents the backend configuration that the model runs with.
	//
	// Can be used in conjunction with the `seed` request parameter to understand when
	// backend changes have been made that might impact determinism.
	SystemFingerprint string `json:"system_fingerprint,omitempty"`

	// ServiceTier is the service tier of the response.
	// e.g. "free", "standard", "premium"
	ServiceTier string `json:"service_tier,omitempty"`

	// Error is the error information, will present if request to llm service failed with status >= 400.
	Error *ResponseError `json:"error,omitempty"`
}

// Choice represents a choice in the response.
// Choice represents a choice in the response.
type Choice struct {
	// Index is the index of the choice in the list of choices.
	Index int `json:"index"`

	// Message is the message content, will present if stream is false
	Message *Message `json:"message,omitempty"`

	// Delta is the stream event content, will present if stream is true
	Delta *Message `json:"delta,omitempty"`

	// FinishReason is the reason the model stopped generating tokens.
	// e.g. "stop", "length", "content_filter", "function_call", "tool_calls"
	FinishReason *string `json:"finish_reason,omitempty"`

	Logprobs *LogprobsContent `json:"logprobs,omitempty"`
}

// LogprobsContent represents logprobs information.
type LogprobsContent struct {
	Content []TokenLogprob `json:"content"`
}

// TokenLogprob represents logprob for a token.
type TokenLogprob struct {
	Token       string       `json:"token"`
	Logprob     float64      `json:"logprob"`
	Bytes       []int        `json:"bytes,omitempty"`
	TopLogprobs []TopLogprob `json:"top_logprobs,omitempty"`
}

// TopLogprob represents top alternative tokens.
type TopLogprob struct {
	Token   string  `json:"token"`
	Logprob float64 `json:"logprob"`
	Bytes   []int   `json:"bytes,omitempty"`
}

type ResponseMeta struct {
	ID    string `json:"id"`
	Usage *Usage `json:"usage"`
}

// Usage Represents the total token usage per request to OpenAI.
type Usage struct {
	PromptTokens            int                      `json:"prompt_tokens"`
	CompletionTokens        int                      `json:"completion_tokens"`
	TotalTokens             int                      `json:"total_tokens"`
	PromptTokensDetails     *PromptTokensDetails     `json:"prompt_tokens_details"`
	CompletionTokensDetails *CompletionTokensDetails `json:"completion_tokens_details"`
}

// CompletionTokensDetails Breakdown of tokens used in a completion.
type CompletionTokensDetails struct {
	AudioTokens              int `json:"audio_tokens"`
	ReasoningTokens          int `json:"reasoning_tokens"`
	AcceptedPredictionTokens int `json:"accepted_prediction_tokens"`
	RejectedPredictionTokens int `json:"rejected_prediction_tokens"`
}

// PromptTokensDetails Breakdown of tokens used in the prompt.
type PromptTokensDetails struct {
	AudioTokens  int `json:"audio_tokens"`
	CachedTokens int `json:"cached_tokens"`
}

// ResponseError represents an error response.
type ResponseError struct {
	StatusCode int         `json:"-"`
	Detail     ErrorDetail `json:"error"`
}

func (e ResponseError) Error() string {
	return fmt.Sprintf("error: %s, code: %s, type: %s, param: %s, request_id: %s", e.Detail.Message, e.Detail.Code, e.Detail.Type, e.Detail.Param, e.Detail.RequestID)
}

// ErrorDetail represents error details.
type ErrorDetail struct {
	Code      string `json:"code,omitempty"`
	Message   string `json:"message"`
	Type      string `json:"type"`
	Param     string `json:"param,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}
