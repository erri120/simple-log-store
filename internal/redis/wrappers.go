package redis

import (
	"context"
	"fmt"
	"log/slog"
)

func (s *Service) set(namespace string, key string, value string, ctx context.Context) error {
	redisKey := fmt.Sprintf("%s:%s", namespace, key)
	if err := s.client.Set(ctx, redisKey, value, s.logRetentionDuration).Err(); err != nil {
		s.logger.Error("failed to set value for key", slog.String("key", redisKey), slog.String("value", value))
		return err
	}

	return nil
}
