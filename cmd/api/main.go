package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/vsayfb/gig-platform-core-service/config"
	"github.com/vsayfb/gig-platform-core-service/internal/user"
	"github.com/vsayfb/gig-platform-core-service/internal/user/auth"
	"github.com/vsayfb/gig-platform-core-service/pkg/database"
	"github.com/vsayfb/gig-platform-core-service/pkg/google"
	"github.com/vsayfb/gig-platform-core-service/pkg/jwt"
	"github.com/vsayfb/gig-platform-core-service/pkg/logger"
	"github.com/vsayfb/gig-platform-core-service/pkg/middleware"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Load()

	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	logger.Init(cfg.Env)

	slog.Info("starting server", "env", cfg.Env, "port", cfg.Server.Port)

	db, err := database.NewPool(ctx, cfg.DB.DSN())

	if err != nil {
		slog.Error("failed to connect to db", "err", err)

		os.Exit(1)
	}

	defer db.Close()

	slog.Info("db connected")

	googleVerifier, err := google.NewVerifier(ctx, cfg.Google.ClientID)

	if err != nil {
		slog.Error("failed to initialize google verifier", "err", err)

		os.Exit(1)
	}

	slog.Info("google oidc verifier initialized")

	jwtManager := jwt.NewManager(cfg.JWT.Secret, cfg.JWT.Expiration)

	userRepo := user.NewUserRepository(db)
	authRepo := auth.NewUserAuthRepository(db)

	userService := user.NewUserService(userRepo)
	authService := auth.NewUserAuthService(authRepo, userRepo, *googleVerifier, jwtManager, db)

	userHandler := user.NewUserHandler(userService)
	authHandler := auth.NewUserAuthHandler(authService)

	r := chi.NewRouter()
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.RequestID)

	r.Group(func(r chi.Router) {
		authHandler.RegisterRoutes(r)
	})

	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(jwtManager))
		userHandler.RegisterRoutes(r)
	})

	slog.Info("server ready", "port", cfg.Server.Port)

	if err := http.ListenAndServe(":"+cfg.Server.Port, r); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
