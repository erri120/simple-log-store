package redis

import (
	"context"
	"github.com/oklog/ulid/v2"
	"simple-log-store/internal/logs"
	"simple-log-store/internal/utils"
	"time"
)

// namespace contains all staged log files where the value is the staging time in UTC
const stagedLogsNamespace = "stagedLogs"

// namespace contains all log bundles where the value is a JSON array containing all referenced log file IDs
const logBundlesNamespace = "logBundles"

func (s *Service) StageLogFile(ctx context.Context, id logs.LogFileId) error {
	now := time.Now().UTC()
	dateTimeString := now.Format(time.RFC3339Nano)

	if err := s.set(stagedLogsNamespace, id.String(), dateTimeString, ctx); err != nil {
		return err
	}

	return nil
}

func (s *Service) CreateLogBundle(ctx context.Context, logFileIds []logs.LogFileId) (logs.LogBundleId, error) {
	bundleId := ulid.Make()

	encoded, err := logs.EncodeIds(logFileIds)
	if err != nil {
		s.logger.Error("failed to encode IDs", utils.ErrAttr(err))
		return bundleId, err
	}

	if err := s.set(logBundlesNamespace, bundleId.String(), encoded, ctx); err != nil {
		return bundleId, err
	}

	return bundleId, nil
}
