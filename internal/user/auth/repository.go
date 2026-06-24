package auth

import "context"

type UserAuthRepository interface {
	Save(ctx context.Context, auth *UserAuth) error
	FindByGoogleSub(ctx context.Context, googleSub string) (*UserAuth, error)
	FindByPhoneHmac(ctx context.Context, hmac string) (*UserAuth, error)
}
