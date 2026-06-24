package auth

type UserAuthService struct {
	authRepo UserAuthRepository
}

func NewUserAuthService(repo UserAuthRepository) *UserAuthService {
	return &UserAuthService{
		authRepo: repo,
	}
}
