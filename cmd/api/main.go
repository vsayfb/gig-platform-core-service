package main

import (
	"context"
	"log"

	"github.com/vsayfb/gig-platform-core-service/config"
	"github.com/vsayfb/gig-platform-core-service/pkg/database"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Load()

	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	_, err = database.NewPool(ctx, cfg.DB.DSN())

	if err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}

}
