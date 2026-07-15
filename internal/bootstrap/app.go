package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vsayfb/gig-platform-core-service/config"
	"github.com/vsayfb/gig-platform-core-service/pkg/grpcserver"
)

type App struct {
	cfg         *config.Config
	db          *pgxpool.Pool
	httpSrv     *http.Server
	grpcService *grpcserver.Server

	shutdownTelemetry func(context.Context) error
}

func NewApp(ctx context.Context, cfg *config.Config) (*App, error) {
	logHandler := initLogger(cfg)

	slog.Info("starting app", "env", cfg.Env)

	db, err := newDatabase(ctx, cfg)

	if err != nil {
		return nil, fmt.Errorf("database: %w", err)
	}

	googleVerifier, err := newGoogleVerifier(ctx, cfg)

	if err != nil {
		db.Close()
		return nil, fmt.Errorf("google verifier: %w", err)
	}

	slog.Info("google oidc verifier initialized")

	jwtManager := newJWTManager(cfg)

	sqsPublisher, err := newSQSPublisher(ctx, cfg)

	if err != nil {
		db.Close()
		return nil, fmt.Errorf("sqs publisher: %w", err)
	}

	slog.Info("sqs client initialized")

	repos := newRepositories(db)
	svcs := newServices(repos, googleVerifier, jwtManager, db)
	hdlrs := newHandlers(svcs, sqsPublisher)

	slog.Info("dependencies injected")

	shutdownTelemetry, err := initTelemetry(ctx, cfg)

	if err != nil {
		db.Close()
		return nil, fmt.Errorf("telemetry: %w", err)
	}

	slog.SetDefault(withTracing(logHandler))

	router := newRouter(cfg, hdlrs, jwtManager)
	grpcService := newGRPCServer(cfg, svcs)

	httpSrv := &http.Server{
		Addr:    ":" + cfg.REST.Port,
		Handler: router,
	}

	return &App{
		cfg:               cfg,
		db:                db,
		httpSrv:           httpSrv,
		grpcService:       grpcService,
		shutdownTelemetry: shutdownTelemetry,
	}, nil
}

func (a *App) Run() {
	go func() {
		if err := a.grpcService.Start(); err != nil {
			slog.Error("grpc failed", "err", err)
			os.Exit(1)
		}
	}()

	go func() {
		slog.Info("grpc ready", "port", a.cfg.GRPC.Port)
		slog.Info("rest ready", "port", a.cfg.REST.Port)

		if err := a.httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("rest failed", "err", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down")

	a.Shutdown()
}

func (a *App) Shutdown() {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	a.grpcService.Stop()

	if err := a.httpSrv.Shutdown(shutdownCtx); err != nil {
		slog.Error("failed to close server's http connection", "err", err)
	}

	telemetryCtx, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()

	if err := a.shutdownTelemetry(telemetryCtx); err != nil {
		slog.Error("telemetry shutdown error", "error", err)
	}

	a.db.Close()

	slog.Info("shutdown complete")
}
