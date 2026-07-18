// Package migrations embeds SQL database migration files.
package migrations

import "embed"

// FS contains all SQL migration files in this directory.
//
//go:embed *.sql
var FS embed.FS
