package internal

import "github.com/oklog/ulid/v2"

type LogBundleId = ulid.ULID

func CreateLogBundle(logFileIds []LogFileId) (LogBundleId, error) {
	bundleId := ulid.Make()

	return bundleId, nil
}
