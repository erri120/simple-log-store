package storage

import (
	"fmt"
	"log/slog"
	"os"
)

func (s *Service) createDirectory(directoryPath string) error {
	fileInfo, err := os.Stat(directoryPath)
	if err != nil {
		if os.IsNotExist(err) {
			err := os.MkdirAll(directoryPath, s.directoryPermissions)
			if err != nil {
				return fmt.Errorf("failed to create directory at `%s`: %w", directoryPath, err)
			}

			s.logger.Info("created new directory", slog.String("directoryPath", directoryPath))
			return nil
		}

		return fmt.Errorf("unexpected error: %w", err)
	}

	if !fileInfo.IsDir() {
		return fmt.Errorf("exptected a directory at `%s` but found a file instead", directoryPath)
	}

	s.logger.Info("using existing directory", slog.String("directoryPath", directoryPath))
	return nil
}

type moveFileFunc func(string, string) error

func noMove(_ string, _ string) error {
	// NOTE(erri120): noop when from and to are the same
	return nil
}

func (s *Service) hardlinkFile(from string, to string) error {
	if err := os.Link(from, to); err != nil {
		return fmt.Errorf("failed to create a hardlink from `%s` to `%s`: %w", from, to, err)
	}

	if err := os.Remove(from); err != nil {
		return fmt.Errorf("failed to remove file `%s` after creating a hardlink to `%s`: %w", from, to, err)
	}

	return nil
}

func (s *Service) copyFile(from string, to string) error {
	fileInfo, err := os.Stat(from)
	if err != nil {
		return fmt.Errorf("failed to stat file `%s`: %w", from, err)
	}

	if fileInfo.IsDir() {
		return fmt.Errorf("expected `%s` to be a file but found a directory", from)
	}

	permissions := fileInfo.Mode().Perm()

	fromFile, err := os.Open(from)
	defer func(logger *slog.Logger, file *os.File, path string) {
		if err := file.Close(); err != nil {
			logger.Error("failed to close file", slog.String("path", path))
			return
		}

		if err := os.Remove(path); err != nil {
			logger.Error("failed to remove original file", slog.String("path", path))
			return
		}
	}(s.logger, fromFile, from)

	if err != nil {
		return fmt.Errorf("failed to open file `%s`: %w", from, err)
	}

	toFile, err := os.OpenFile(to, os.O_CREATE|os.O_WRONLY|os.O_EXCL, permissions)
	defer func(logger *slog.Logger, file *os.File, path string) {
		if err := file.Close(); err != nil {
			logger.Error("failed to close file", slog.String("path", path))
			return
		}
	}(s.logger, toFile, to)

	if err != nil {
		return fmt.Errorf("failed to open file `%s`: %w", to, err)
	}

	_, err = fromFile.WriteTo(toFile)
	if err != nil {
		return fmt.Errorf("failed to write contents from `%s` to `%s`: %w", from, to, err)
	}

	return nil
}
