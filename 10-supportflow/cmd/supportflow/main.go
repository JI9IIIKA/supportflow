package main

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"supportflow/internal/platform"
	"supportflow/migrations"
	"supportflow/projects/supportflow"
)

func main() {
	platform.Serve("supportflow", supportflow.Register, func(ctx context.Context, pool *pgxpool.Pool) error { return migrations.Apply(ctx, pool, "supportflow") })
}
