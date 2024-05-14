package database

import (
	"context"
	"fmt"
	"github.com/oklog/ulid/v2"
	"github.com/redis/go-redis/v9"
	"time"
)

func AddLogFile(client *redis.Client, ctx context.Context, expiration time.Duration) {
	id := ulid.Make()
	key := fmt.Sprintf("logfile:%s", id)

	_ = client.SetNX(ctx, key, "", expiration)
}
