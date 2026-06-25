package google

import "context"

type TokenVerifier interface {
	Verify(ctx context.Context, idToken string) (*Claims, error)
}
