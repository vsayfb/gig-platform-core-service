package auth

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vsayfb/gig-platform-core-service/internal/user"
	"github.com/vsayfb/gig-platform-core-service/internal/user/reputation"
	"github.com/vsayfb/gig-platform-core-service/pkg/google"
	"github.com/vsayfb/gig-platform-core-service/pkg/jwt"
)

type AuthResult struct {
	Token string
	User  *user.User
}

type UserAuthService struct {
	authRepo          UserAuthRepository
	userRepo          user.UserRepository
	reputationService *reputation.UserReputationService
	tokenVerifier     google.Verifier
	jwtManager        *jwt.Manager
	db                *pgxpool.Pool
}

func NewUserAuthService(
	authRepo UserAuthRepository,
	userRepo user.UserRepository,
	reputationService *reputation.UserReputationService,
	verifier google.Verifier,
	jwtManager *jwt.Manager,
	db *pgxpool.Pool,
) *UserAuthService {
	return &UserAuthService{
		authRepo:          authRepo,
		userRepo:          userRepo,
		reputationService: reputationService,
		tokenVerifier:     verifier,
		jwtManager:        jwtManager,
		db:                db,
	}
}

// handles both registration and login via OIDC.
// if the user exists, login. if not, register then login.
func (s *UserAuthService) GoogleLogin(ctx context.Context, idToken string) (*AuthResult, error) {
	claims, err := s.tokenVerifier.Verify(ctx, idToken)

	if err != nil {
		return nil, fmt.Errorf("invalid google token: %w", err)
	}

	existing, err := s.authRepo.FindByGoogleSub(ctx, claims.Sub)

	if err == nil {
		u, err := s.userRepo.FindByID(ctx, existing.UserID)

		if err != nil {
			// user's auth record exists but profile doesn't
			slog.Error("user auth record orphaned", "google_sub", claims.Sub, "user_id", claims.Sub)

			return nil, fmt.Errorf("failed to fetch user: %w", err)
		}

		return s.issueToken(u)
	}

	u, err := s.register(ctx, claims)

	if err != nil {
		slog.Error("failed to register user", "err", err)

		return nil, fmt.Errorf("failed to register user: %w", err)
	}

	slog.Info("new user registered", "user_id", u.ID)

	return s.issueToken(u)
}

func (s *UserAuthService) register(ctx context.Context, claims *google.Claims) (*user.User, error) {
	tx, err := s.db.Begin(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer tx.Rollback(ctx)

	newUser := user.NewUser(claims.Name, claims.Email)
	createdUser, err := s.userRepo.Save(ctx, newUser)

	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	authRecord := NewUserAuth(createdUser.ID, claims.Sub, nil, nil)

	if _, err := s.authRepo.Save(ctx, authRecord); err != nil {
		return nil, fmt.Errorf("failed to create user auth: %w", err)
	}

	if _, err := s.reputationService.Initialize(ctx, createdUser.ID); err != nil {
		return nil, fmt.Errorf("failed to initialize reputation: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return createdUser, nil
}

func (s *UserAuthService) issueToken(u *user.User) (*AuthResult, error) {
	token, err := s.jwtManager.Generate(u.ID.String())

	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &AuthResult{Token: token, User: u}, nil
}
