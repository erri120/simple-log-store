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
