package storage

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"simple-log-store/internal/logs"
	"simple-log-store/internal/utils"
)

func (s *Service) getStagingPath(id logs.LogFileId) string {
	return filepath.Join(s.stagingPath, id.String())
}

func (s *Service) getStoragePath(id logs.LogFileId) string {
	return filepath.Join(s.storagePath, id.String())
}

type FileTooLarge struct {
	Limit  uint64
	Actual uint64
}

func (f FileTooLarge) Error() string {
	return fmt.Sprintf("expected file size to be less than `%d` bytes but received `%d` bytes", f.Limit, f.Actual)
}

func (s *Service) StageLogFile(id logs.LogFileId, reader io.Reader, maxFileSize uint64) error {
	logFilePath := s.getStagingPath(id)
	logger := s.logger.With(slog.String("logFilePath", logFilePath), slog.String("logFileId", id.String()))

	tmp := false
	shouldCleanup := &tmp

	defer func(logger *slog.Logger, shouldCleanup *bool, logFilePath *string) {
		if !*shouldCleanup {
			return
		}

		logger.Info("starting cleanup of log file in staging process")

		err := os.Remove(*logFilePath)
		if err != nil {
			if os.IsNotExist(err) {
				logger.Error("log file that's supposed to be cleaned up doesn't exist anymore", utils.ErrAttr(err))
				return
			}

			logger.Error("failed to cleanup log file, the file might still exist on disk", utils.ErrAttr(err))
			return
		}

		logger.Info("finished cleanup successfully")
	}(logger, shouldCleanup, &logFilePath)

	logger.Info("begin staging log file")

	file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_RDWR|os.O_EXCL, s.filePermissions)
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	if err != nil {
		*shouldCleanup = true
		logger.Error("failed to open file for writing")
		return fmt.Errorf("failed to open file for writing: %w", err)
	}

	wrappedReader := io.LimitReader(reader, int64(maxFileSize))
	n, err := file.ReadFrom(wrappedReader)
	if err != nil {
		*shouldCleanup = true
		if errors.Is(err, io.EOF) {
			logger.Error("file too big to upload", slog.Int64("bytes", n))
			return FileTooLarge{
				Limit:  maxFileSize,
				Actual: uint64(n),
			}
		}

		return fmt.Errorf("unexpected error while writing to file: %w", err)
	}

	logger.Info("successfully staged log file", slog.Int64("bytes", n))
	return nil
}

func (s *Service) OpenLogFile(logFileId logs.LogFileId) (*os.File, error) {
	// TODO: use storage path instead of staging
	logFilePath := s.getStagingPath(logFileId)

	file, err := os.Open(logFilePath)

	if err != nil {
		s.logger.Error("failed to open log file for reading", utils.ErrAttr(err))
		return nil, err
	}

	return file, nil
}
