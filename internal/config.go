package internal

type AppConfig struct {
	Port uint16 `env:"PORT, default=3000"`

	SingleFileSizeLimit int64 `env:"SINGLE_FILE_SIZE_LIMIT, default=1048576"`
	MaxFileCount        int64 `env:"MAX_FILE_COUNT_PER_BUNDLE, default=5"`
}
