package internal

import (
	"context"
	"fmt"
	"github.com/go-chi/httplog/v2"
	"github.com/oklog/ulid/v2"
	"github.com/redis/go-redis/v9"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

type LogFileId = ulid.ULID

type LogRepo struct {
	RedisClient *redis.Client
	Config      *AppConfig
}

func (repo *LogRepo) StoreLogFile(bytes []byte) (LogFileId, error) {
	id := ulid.Make()
	filePath := filepath.Join(repo.Config.StoragePath, fmt.Sprintf("%s.log", id.String()))
	err := os.WriteFile(filePath, bytes, 0660)
	if err != nil {
		slog.With(slog.String("filePath", filePath)).Error("error writing log to file", httplog.ErrAttr(err))

		_, statErr := os.Stat(filePath)
		if !os.IsNotExist(statErr) {
			_ = os.Remove(filePath)
		}

		return id, err
	}

	slog.With(slog.String("filePath", filePath), slog.Int("bytes", len(bytes))).Info("wrote log to file")
	return id, nil
}

func (repo *LogRepo) AddLogFile(client *redis.Client, ctx context.Context, expiration time.Duration) {
	id := ulid.Make()
	key := fmt.Sprintf("logfile:%s", id)

	_ = client.SetNX(ctx, key, "", expiration)
}
