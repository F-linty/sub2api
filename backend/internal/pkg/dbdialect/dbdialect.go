// Package dbdialect exposes the active database dialect as process-global state so
// that low-level helpers in different packages (repository, service) can branch on
// PostgreSQL vs CockroachDB without an import cycle or threading the value through
// every call site.
//
// It is set once at startup (see repository.InitEnt) from the resolved dialect and
// read on hot paths via IsCockroach. The default is PostgreSQL, so code that never
// calls SetCockroach (unit tests, single-dialect deployments) behaves as before.
package dbdialect

import "sync/atomic"

var cockroach atomic.Bool

// SetCockroach records whether the process is connected to CockroachDB.
func SetCockroach(v bool) { cockroach.Store(v) }

// IsCockroach reports whether the active database is CockroachDB.
func IsCockroach() bool { return cockroach.Load() }
