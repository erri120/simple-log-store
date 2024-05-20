package redis

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"log/slog"
	"simple-log-store/internal/config"
	"simple-log-store/internal/utils"
	"time"
)

type Service struct {
	logger *slog.Logger

	client               *redis.Client
	logRetentionDuration time.Duration
}

func CreateService(appConfig *config.AppConfig, logger *slog.Logger) (*Service, error) {
	opt, err := redis.ParseURL(appConfig.RedisConnectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis URL: %w", err)
	}

	redisClient := redis.NewClient(opt)

	service := &Service{
		logger:               logger.With(slog.String("service", "redis")),
		client:               redisClient,
		logRetentionDuration: appConfig.LogRetentionDuration,
	}

	return service, nil
}

func (s *Service) Ping() error {
	timeout, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	if err := s.client.Ping(timeout).Err(); err != nil {
		return fmt.Errorf("failed to ping redis: %w", err)
	}

	return nil
}

func (s *Service) Close() {
	if err := s.client.Close(); err != nil {
		s.logger.Error("failed to close redis client", utils.ErrAttr(err))
	}
}
