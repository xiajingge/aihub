package config

import (
	"errors"
	"fmt"
	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"strings"
)

var Module = fx.Module("config", fx.Provide(NewConfig))
var Config_Module = Module

type Config struct {
	fx.Out

	APIServer ServerConfig `conf:"server" yaml:"server" json:"server"`
	DB        DBConfig     `conf:"db" yaml:"db" json:"db"`
	Logger    LoggerConfig `conf:"logger" yaml:"logger" json:"logger"`
}

func NewConfig() (Config, error) {
	v := viper.New()

	// Set config file name and paths
	v.SetConfigName("config")
	v.SetConfigType("yml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")
	v.AddConfigPath("./conf")
	v.AddConfigPath("/etc/axonhub/")
	v.AddConfigPath("$HOME/.axonhub")

	// Enable environment variable support
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Set default values
	setDefaults(v)

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			return Config{}, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found, use defaults and environment variables
	}

	// Unmarshal config
	var config Config
	if err := v.Unmarshal(&config, func(dc *mapstructure.DecoderConfig) {
		dc.TagName = "conf"
	}); err != nil {
		return Config{}, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return config, nil
}

func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.port", 8090)
	v.SetDefault("server.name", "AiHub")
	v.SetDefault("server.base_path", "")
	v.SetDefault("server.request_timeout", "30s")
	v.SetDefault("server.llm_request_timeout", "300s")
	v.SetDefault("server.trace.trace_header", "AI-Trace-Id")
	v.SetDefault("server.debug", false)

	// Database defaults
	v.SetDefault("db.dialect", "mysql")
	v.SetDefault("db.dsn", "root:abc123@tcp(127.0.0.1:3306)/aihub?parseTime=True")
	v.SetDefault("db.debug", false)

	// Loggger default
	v.SetDefault("logger.level", "debug")
	v.SetDefault("logger.encoding", "json")
	v.SetDefault("logger.output_paths", []string{"stdout", "./logs/app.log"})

	// GC defaults
	v.SetDefault("gc.cron", "0 2 * * *") // Daily at 2:00 AM
}
