package auth

import "embed"

// MigrationsFS contains the database migration files
//
//go:embed migrations/*.sql
var MigrationsFS embed.FS
