package internal

import (
	"context"
	"fmt"
	"github.com/oklog/ulid/v2"
	"github.com/redis/go-redis/v9"
	"time"
)

type LogFileId = ulid.ULID

func StoreLogFile(bytes []byte) (LogFileId, error) {
	id := ulid.Make()

	return id, nil
}

func AddLogFile(client *redis.Client, ctx context.Context, expiration time.Duration) {
	id := ulid.Make()
	key := fmt.Sprintf("logfile:%s", id)

	_ = client.SetNX(ctx, key, "", expiration)
}
