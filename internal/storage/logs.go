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
	"time"
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

func (s *Service) StoreLogFiles(logFileIds []logs.LogFileId) {
	for _, logFileId := range logFileIds {
		stagingPath := s.getStagingPath(logFileId)
		storagePath := s.getStoragePath(logFileId)

		if err := s.moveFileFunc(stagingPath, storagePath); err != nil {
			s.logger.Error("failed to store log file", slog.String("stagingPath", stagingPath), slog.String("storagePath", storagePath), utils.ErrAttr(err))
		}
	}
}

func (s *Service) OpenLogFile(logFileId logs.LogFileId) (*os.File, error) {
	logFilePath := s.getStoragePath(logFileId)

	file, err := os.Open(logFilePath)

	if err != nil {
		s.logger.Error("failed to open log file for reading", slog.String("logFilePath", logFilePath), utils.ErrAttr(err))
		return nil, err
	}

	return file, nil
}

func (s *Service) DeleteLogFile(logFileId logs.LogFileId) error {
	logFilePath := s.getStoragePath(logFileId)

	if err := os.Remove(logFilePath); err != nil {
		s.logger.Error("failed to remove log file", slog.String("logFilePath", logFilePath), utils.ErrAttr(err))
		return err
	}

	return nil
}

func (s *Service) RemoveOldLogFiles(before time.Time) error {
	directoryPath := s.storagePath

	s.logger.Info("begin removing old log files")
	defer func(logger *slog.Logger) {
		logger.Info("finished removing old log files")
	}(s.logger)

	directoryEntries, err := os.ReadDir(directoryPath)
	if err != nil {
		s.logger.Error("error while reading directory", slog.String("directoryPath", directoryPath), utils.ErrAttr(err))
		if len(directoryEntries) == 0 {
			return err
		}
	}

	for _, directoryEntry := range directoryEntries {
		filePath := filepath.Join(directoryPath, directoryEntry.Name())
		fileInfo, err := directoryEntry.Info()
		if err != nil {
			s.logger.Error("failed to get info of file", slog.String("filePath", filePath), utils.ErrAttr(err))
			continue
		}

		modificationTime := fileInfo.ModTime()
		if !modificationTime.Before(before) {
			continue
		}

		s.logger.Info("removing old log file", slog.String("filePath", filePath))
		if err := os.Remove(filePath); err != nil {
			s.logger.Error("failed to remove old log file", slog.String("filePath", filePath), utils.ErrAttr(err))
			continue
		}
	}

	return nil
}
