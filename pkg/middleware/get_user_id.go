package middleware

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

type contextKey struct{}

var UserIDContextKey = contextKey{}

var ErrUserIDNotFound = errors.New("user id not found in context")

func UserIDFromContext(ctx context.Context) (uuid.UUID, error) {
	userID, ok := ctx.Value(UserIDContextKey).(uuid.UUID)

	if !ok {
		return uuid.Nil, ErrUserIDNotFound
	}

	return userID, nil
}
