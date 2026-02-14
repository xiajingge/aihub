package config

import "time"

type ServerConfig struct {
	Port        int           `yaml:"port" json:"port"`
	Name        string        `yaml:"name" json:"name"`
	BasePath    string        `yaml:"base_path" json:"base_path"`
	ReadTimeout time.Duration `yaml:"read_timeout" json:"read_timeout"`

	// WriteTimeout is the maximum duration for writing the response.
	WriteTimeout time.Duration `yaml:"write_timeout" json:"write_timeout"`

	// RequestTimeout is the maximum duration for processing a request.
	RequestTimeout time.Duration `yaml:"request_timeout" json:"request_timeout"`

	// LLMRequestTimeout is the maximum duration for processing a request to LLM.
	LLMRequestTimeout time.Duration `yaml:"llm_request_timeout" json:"llm_request_timeout"`

	Debug bool `conf:"debug" yaml:"debug" json:"debug"`
}

type DBConfig struct {
	Dialect string `conf:"dialect" yaml:"dialect" json:"dialect"`
	DSN     string `conf:"dsn" yaml:"dsn" json:"dsn"`
	Debug   bool   `conf:"debug" yaml:"debug" json:"debug"`
}

type LoggerConfig struct {
	Level       string   `conf:"level" yaml:"level" json:"level"`
	Encoding    string   `conf:"encoding" yaml:"encoding" json:"encoding"`
	OutputPaths []string `conf:"output_paths" yaml:"output_paths" json:"output_paths"`
}
