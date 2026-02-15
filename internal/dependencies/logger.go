package dependencies

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/xiajignge/aihub/config"
	"github.com/xiajingge/logger"
)

func NewLogger(cfg config.LoggerConfig) logger.LoggerV1 {
	outputPaths := normalizeOutputPaths(cfg.OutputPaths)

	c := logger.Config{
		Level:            cfg.Level,
		Encoding:         cfg.Encoding,
		OutputPaths:      outputPaths,
		ErrorOutputPaths: []string{"stderr"},
	}

	return logger.InitWithConfig(c)
}

func normalizeOutputPaths(paths []string) []string {
	if len(paths) == 0 {
		paths = []string{"stdout", "./logs/app.log"}
	}

	normalized := make([]string, 0, len(paths))
	for _, path := range paths {
		p := strings.TrimSpace(path)
		if p == "" {
			continue
		}

		switch p {
		case "stdout", "stderr":
			normalized = append(normalized, p)
		default:
			dir := filepath.Dir(p)
			if dir != "" && dir != "." {
				_ = os.MkdirAll(dir, 0o755)
			}
			normalized = append(normalized, p)
		}
	}

	if len(normalized) == 0 {
		return []string{"stdout", "./logs/app.log"}
	}

	return normalized
}
