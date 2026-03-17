package raid_lobby

import "embed"

//go:embed migrations/*.sql
var MigrationsFS embed.FS
