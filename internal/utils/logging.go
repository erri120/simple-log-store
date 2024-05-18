package utils

import (
	"log/slog"
)

func ErrAttr(err error) slog.Attr {
	return slog.String("err", err.Error())
}
