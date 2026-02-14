package dependencies

import (
	"github.com/gaohao-creator/executors"
	"github.com/xiajingge/logger"
)

type ErrorHandler struct {
	logger logger.LoggerV1
}

func NewErrorHandler(logger logger.LoggerV1) *ErrorHandler {
	return &ErrorHandler{logger: logger}
}

func (h *ErrorHandler) CatchError(runnable executors.Runnable, err error) {
	h.logger.Error("executor error", logger.Error(err))
}

func NewExecutors(handler *ErrorHandler) executors.ScheduledExecutor {
	return executors.NewPoolScheduleExecutor(
		executors.WithErrorHandler(handler),
		executors.WithMaxConcurrent(10),
	)
}
