package main

import (
	"context"
	"log"
	"log/slog"
	"os"

	"github.com/vsayfb/gig-platform-core-service/config"
	"github.com/vsayfb/gig-platform-core-service/pkg/database"
	"github.com/vsayfb/gig-platform-core-service/pkg/logger"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Load()

	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	logger.Init(cfg.Env)

	db, err := database.NewPool(ctx, cfg.DB.DSN())

	if err != nil {
		slog.Error("failed to connect to db", "err", err)
		os.Exit(1)
	}

	defer db.Close()

	slog.Info("db connected")
}
