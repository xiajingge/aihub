package dependencies

import (
	"github.com/xiajignge/aihub/config"
	"github.com/xiajingge/logger"
)

func NewLogger(cfg config.LoggerConfig) logger.LoggerV1 {
	c := logger.Config{
		Level:       cfg.Level,       // 设置日志级别
		Encoding:    cfg.Encoding,    // 使用 JSON 格式 ("json" 或 "console")
		OutputPaths: cfg.OutputPaths, // 日志文件路径
	}

	return logger.InitWithConfig(c)
}
