package user

import "context"

type UserRepository interface {
	Save(ctx context.Context, user *User) error
}
