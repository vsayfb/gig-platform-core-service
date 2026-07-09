package bootstrap

import (
	"context"
	"fmt"

	"github.com/vsayfb/gig-platform-core-service/config"
	"github.com/vsayfb/gig-platform-core-service/pkg/google"
	"github.com/vsayfb/gig-platform-core-service/pkg/jwt"
)

func newGoogleVerifier(ctx context.Context, cfg *config.Config) (*google.Verifier, error) {
	v, err := google.NewVerifier(ctx, cfg.Google.ClientID)
	if err != nil {
		return nil, fmt.Errorf("initialize google verifier: %w", err)
	}
	return v, nil
}

func newJWTManager(cfg *config.Config) *jwt.Manager {
	return jwt.NewManager(cfg.JWT.Secret, cfg.JWT.Expiration)
}
