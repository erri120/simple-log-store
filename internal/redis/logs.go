package redis

import (
	"context"
	"errors"
	"fmt"
	"github.com/oklog/ulid/v2"
	"github.com/redis/go-redis/v9"
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

var ErrNotFound = errors.New("item not found")

func (s *Service) GetLogBundle(ctx context.Context, logBundleId logs.LogBundleId) ([]logs.LogFileId, error) {
	cmd := s.client.Get(ctx, fmt.Sprintf("%s:%s", logBundlesNamespace, logBundleId.String()))
	bytes, err := cmd.Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, fmt.Errorf("unable to find log bundle with ID `%s`: %w", logBundleId.String(), ErrNotFound)
		}

		return nil, fmt.Errorf("failed to get bytes for log bundle with ID `%s`: `%w`", logBundleId.String(), err)
	}

	logFileIds, err := logs.DecodeIds(bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to decode log file IDs for bundle `%s`: `%w`", logBundleId.String(), err)
	}

	return logFileIds, nil
}
