package sql

import "embed"

// MigrationsFS exposes the embedded migration files to other internal packages
//
//go:embed migrations/*.sql
var MigrationsFS embed.FS
