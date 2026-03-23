package query

import "errors"

var (
	errInvalidBody      = errors.New("invalid request body")
	errOrgNotFound      = errors.New("organization not found")
	errRepoNotFound     = errors.New("repository not found")
	errNoAIKey          = errors.New("no AI provider key configured — add one in Settings")
	errSnapshotNotFound = errors.New("no snapshot available")
	errSnapshotOutdated = errors.New("snapshot outdated: re-run zigzag --upload to generate a v2 snapshot")
	errInternal         = errors.New("internal error")
)
