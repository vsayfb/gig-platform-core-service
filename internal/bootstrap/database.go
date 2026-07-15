package bootstrap

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vsayfb/gig-platform-core-service/config"
	"github.com/vsayfb/gig-platform-core-service/pkg/database"
)

func newDatabase(ctx context.Context, cfg *config.Config) (*pgxpool.Pool, error) {
	db, err := database.NewPool(ctx, cfg.DB.DSN())

	if err != nil {
		return nil, fmt.Errorf("connect to db: %w", err)
	}

	slog.Info("db connected")

	if err := database.RunMigrations(cfg.DB.URL()); err != nil {
		db.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	return db, nil
}
