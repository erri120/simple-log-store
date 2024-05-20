package storage

import (
	"io/fs"
	"log/slog"
	"simple-log-store/internal/config"
)

type Service struct {
	logger *slog.Logger

	stagingPath  string
	storagePath  string
	moveFileFunc moveFileFunc

	directoryPermissions fs.FileMode
	filePermissions      fs.FileMode
}

const defaultDirectoryPermissions = fs.FileMode(0770)
const defaultFilePermissions = fs.FileMode(0660)

func CreateService(appConfig *config.AppConfig, logger *slog.Logger) (*Service, error) {
	service := &Service{
		logger:               logger.With(slog.String("service", "storage")),
		stagingPath:          appConfig.StagingPath,
		storagePath:          appConfig.StoragePath,
		directoryPermissions: fixPermissions(appConfig.DirectoryPermissions, defaultDirectoryPermissions),
		filePermissions:      fixPermissions(appConfig.FilePermissions, defaultFilePermissions),
	}

	if appConfig.UseHardlinks {
		service.moveFileFunc = service.hardlinkFile
	} else {
		service.moveFileFunc = service.copyFile
	}

	if err := service.init(); err != nil {
		return nil, err
	}

	return service, nil
}

func fixPermissions(rawPermissions uint32, defaultPermissions fs.FileMode) fs.FileMode {
	permissions := defaultPermissions
	if rawPermissions != 0 {
		permissions = fs.FileMode(rawPermissions)
	}

	return permissions
}

func (s *Service) init() error {
	if err := s.createDirectory(s.stagingPath); err != nil {
		return err
	}

	if err := s.createDirectory(s.storagePath); err != nil {
		return err
	}

	return nil
}
