package logs

import (
	"fmt"
	"github.com/oklog/ulid/v2"
)

type LogFileId = ulid.ULID

type LogBundleId = ulid.ULID

func EncodeIds(logFileIds []LogFileId) (string, error) {
	if len(logFileIds) < 1 {
		return "", fmt.Errorf("input must have at least one ID")
	}

	outputSize := len(logFileIds) * ulid.EncodedSize
	output := make([]byte, outputSize)

	for i := 0; i < len(logFileIds); i++ {
		sliceStart := i * ulid.EncodedSize
		sliceEnd := (i + 1) * ulid.EncodedSize
		outputSlice := output[sliceStart:sliceEnd]

		cur := logFileIds[i]
		if err := cur.MarshalTextTo(outputSlice); err != nil {
			return "", fmt.Errorf("failed to marshal ulid into output: %w", err)
		}
	}

	return string(output), nil
}

func DecodeIds(input []byte) ([]LogFileId, error) {
	if len(input) < ulid.EncodedSize {
		return nil, fmt.Errorf("input slice of length `%d` is too small, encoding size of ulid is `%d`", len(input), ulid.EncodedSize)
	}

	if len(input)%ulid.EncodedSize != 0 {
		return nil, fmt.Errorf("input slice of length `%d` is not a multiple of `%d`", len(input), ulid.EncodedSize)
	}

	numIds := len(input) / ulid.EncodedSize
	res := make([]LogFileId, numIds)

	for i := 0; i < len(res); i++ {
		sliceStart := i * ulid.EncodedSize
		sliceEnd := (i + 1) * ulid.EncodedSize
		inputSlice := input[sliceStart:sliceEnd]

		var id ulid.ULID
		if err := id.UnmarshalText(inputSlice); err != nil {
			return nil, fmt.Errorf("failed to unmarshal ulid from input: %w", err)
		}

		res[i] = id
	}

	return res, nil
}
