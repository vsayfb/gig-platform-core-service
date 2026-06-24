package google

import (
	"context"
	"fmt"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
)

type Claims struct {
	Sub     string
	Email   string
	Name    string
	Picture string
}

type Verifier struct {
	verifier *oidc.IDTokenVerifier
}

func NewVerifier(ctx context.Context, clientID string) (*Verifier, error) {
	provider, err := oidc.NewProvider(
		ctx,
		"https://accounts.google.com",
	)
	if err != nil {
		return nil, fmt.Errorf("create oidc provider: %w", err)
	}

	return &Verifier{
		verifier: provider.Verifier(&oidc.Config{
			ClientID: clientID,
		}),
	}, nil
}

func (v *Verifier) Verify(
	ctx context.Context,
	idToken string,
) (*Claims, error) {
	token, err := v.verifier.Verify(ctx, idToken)
	if err != nil {
		return nil, fmt.Errorf("verify token: %w", err)
	}

	var c struct {
		Sub        string `json:"sub"`
		Email      string `json:"email"`
		Name       string `json:"name"`
		GivenName  string `json:"given_name"`
		FamilyName string `json:"family_name"`
		Picture    string `json:"picture"`
	}

	if err := token.Claims(&c); err != nil {
		return nil, fmt.Errorf("decode claims: %w", err)
	}

	name := strings.TrimSpace(
		c.GivenName + " " + c.FamilyName,
	)

	if name == "" {
		name = strings.TrimSpace(c.Name)
	}

	return &Claims{
		Sub:     c.Sub,
		Email:   c.Email,
		Name:    name,
		Picture: c.Picture,
	}, nil
}
