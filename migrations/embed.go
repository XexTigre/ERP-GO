package migrations

import "embed"

// Files holds the immutable, checksummed migration history.
//
//go:embed *.sql
var Files embed.FS
