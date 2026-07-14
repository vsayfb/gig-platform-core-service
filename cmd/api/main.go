package main

import (
	"context"
	"log"
	"os"

	"github.com/vsayfb/gig-platform-core-service/config"
	"github.com/vsayfb/gig-platform-core-service/internal/bootstrap"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Load(ctx)

	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	app, err := bootstrap.NewApp(ctx, cfg)

	if err != nil {
		log.Fatalf("failed to initialize app: %v", err)
	}

	app.Run()

	os.Exit(0)
}
