package internal

type AppConfig struct {
	Port uint16 `env:"PORT, default=3000"`

	RedisConnectionString string `env:"REDIS_CONNECTION, default=redis://0.0.0.0:6379"`

	SingleFileSizeLimit int64 `env:"SINGLE_FILE_SIZE_LIMIT, default=1048576"`
	MaxFileCount        int64 `env:"MAX_FILE_COUNT_PER_BUNDLE, default=5"`

	StoragePath string `env:"STORAGE_PATH, required"`
}
