package base

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/xiajignge/aihub/internal/service"
	"github.com/xiajignge/aihub/pkg/httpclient"
	"github.com/xiajingge/logger"
)

type ChatCompletionSSEHandlers struct {
	ChatCompletionService service.ChatCompletionService
	logger                logger.LoggerV1
}

func NewChatCompletionSSEHandlers(
	chatCompletionService service.ChatCompletionService,
	loggerv1 logger.LoggerV1,
) *ChatCompletionSSEHandlers {
	return &ChatCompletionSSEHandlers{
		ChatCompletionService: chatCompletionService,
		logger:                loggerv1,
	}
}

func (handlers *ChatCompletionSSEHandlers) ChatCompletion(c *gin.Context) {
	ctx := c.Request.Context()

	// Use ReadHTTPRequest to parse the request
	genericReq, err := httpclient.ReadHTTPRequest(c.Request)
	if err != nil {
		c.JSON(http.StatusBadRequest, httpclient.Fail("请求信息错误", err.Error()))
		return
	}

	//Let the service handle it.
	result, err := handlers.ChatCompletionService.Process(ctx, genericReq)
	if err != nil {
		handlers.logger.Error("Error processing chat completion", logger.Error(err))
		c.JSON(http.StatusBadRequest, httpclient.Fail("请求信息错误", err.Error()))
		return
	}

	if result.ChatCompletion != nil {
		resp := result.ChatCompletion

		contentType := "application/json"
		if ct := resp.Headers.Get("Content-Type"); ct != "" {
			contentType = ct
		}

		c.Data(resp.StatusCode, contentType, resp.Body)

		return
	}

	if result.ChatCompletionStream != nil {
		defer func() {
			handlers.logger.Debug("Close chat stream")

			err := result.ChatCompletionStream.Close()
			if err != nil {
				handlers.logger.Error("Error closing stream", logger.Error(err))
			}
		}()

		// todo Set appropriate headers based on transformer type
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")

		c.Header("Access-Control-Allow-Origin", "*")

		handlers.writeSSEStream(c, result.ChatCompletionStream)
	}

}

func (h *ChatCompletionSSEHandlers) writeSSEStream(c *gin.Context, stream httpclient.Stream[*httpclient.StreamEvent]) {
	ctx := c.Request.Context()
	clientDisconnected := false

	defer func() {
		if clientDisconnected {
			h.logger.Debug("Client disconnected")
		}
	}()

	clientGone := c.Writer.CloseNotify()

	for {
		select {
		case <-clientGone:
			clientDisconnected = true

			h.logger.Warn("Client disconnected, stopping stream")

			return

		case <-ctx.Done():
			h.logger.Warn("Context done,  stopping stream")

			return
		default:
			if stream.Next() {
				cur := stream.Current()
				c.SSEvent(cur.Type, cur.Data)
				h.logger.Debug("write stream event", logger.String("event", cur.Type))
				h.logger.Info("write stream event", logger.Field{
					Key: "event",
					Val: cur,
				})
				c.Writer.Flush()
			} else {
				if stream.Err() != nil {
					h.logger.Error("Error in stream", logger.Error(stream.Err()))
					c.SSEvent("error", stream.Err())
				}

				return
			}
		}
	}
}
