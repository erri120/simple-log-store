package config

import "time"

type AppConfig struct {
	Port uint16 `env:"PORT, default=3000"`

	RedisConnectionString string `env:"REDIS_CONNECTION, default=redis://0.0.0.0:6379"`

	LogRetentionDuration time.Duration `env:"LOG_RETENTION_DURATION, default=336h"`
	CleanupInterval      time.Duration `env:"CLEANUP_INTERVAL, default=10m"`

	SingleFileSizeLimit uint64 `env:"SINGLE_FILE_SIZE_LIMIT, default=1048576"`
	MaxFileCount        uint16 `env:"MAX_FILE_COUNT_PER_BUNDLE, default=5"`
	UseHardlinks        bool   `env:"USE_HARDLINKS, required"`

	StagingPath string `env:"STAGING_PATH, required"`
	StoragePath string `env:"STORAGE_PATH, required"`

	DirectoryPermissions uint32 `env:"DIRECTORY_UMASK"`
	FilePermissions      uint32 `env:"FILE_MASK"`
}
