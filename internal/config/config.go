package config

import (
	"fmt"

	"github.com/ilyakaznacheev/cleanenv"

	"link-service/internal/logger"
	filesystem "link-service/internal/repository/file_system"
	"link-service/internal/server"
	"link-service/internal/service"
)

type Config struct {
	HTTPServer server.Config
	Storage    filesystem.Config
	Service    service.Config
	Logger     logger.Config
}

func New(path string) (*Config, error) {
	var cfg Config

	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	return &cfg, nil
}
