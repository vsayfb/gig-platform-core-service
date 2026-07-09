package bootstrap

import (
	"context"
	"log/slog"

	"github.com/vsayfb/gig-platform-core-service/config"
	"github.com/vsayfb/gig-platform-core-service/pkg/logger"
	"github.com/vsayfb/gig-platform-core-service/pkg/metrics"
	"github.com/vsayfb/gig-platform-core-service/pkg/telemetry"
	"github.com/vsayfb/gig-platform-core-service/pkg/tracing"
)

func initLogger(cfg *config.Config) slog.Handler {
	return logger.Init(cfg.Env)
}

func withTracing(base slog.Handler) *slog.Logger {
	return slog.New(tracing.NewOTelHandler(base))
}

func initTelemetry(ctx context.Context, cfg *config.Config) (func(context.Context) error, error) {
	shutdown, err := telemetry.Init(ctx, cfg.REST.ServiceName, cfg.REST.OTelCollectorAddr)
	if err != nil {
		return nil, err
	}

	if err := metrics.Register(); err != nil {
		return nil, err
	}

	return shutdown, nil
}
