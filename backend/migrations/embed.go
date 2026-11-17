package migrations

import "embed"

// Files contains the versioned SQL migrations used by API tooling and tests.
//
//go:embed *.sql
var Files embed.FS
