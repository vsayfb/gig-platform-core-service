package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/vsayfb/gig-platform-core-service/internal/user"
	"github.com/vsayfb/gig-platform-core-service/internal/user/auth"
	"github.com/vsayfb/gig-platform-core-service/internal/user/reputation"
	"github.com/vsayfb/gig-platform-core-service/pkg/google"
	"github.com/vsayfb/gig-platform-core-service/pkg/jwt"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

type fakeVerifier struct {
	claims *google.Claims
	err    error
}

func (f *fakeVerifier) Verify(_ context.Context, _ string) (*google.Claims, error) {
	return f.claims, f.err
}

func validClaims() *google.Claims {
	return &google.Claims{
		Sub:   "google-sub-" + uuid.NewString(),
		Email: uuid.NewString() + "@example.com",
		Name:  "Alice",
	}
}

func startPostgres(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()

	pgc, err := postgres.Run(ctx,
		"postgis/postgis:16-3.4",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = pgc.Terminate(ctx) })

	connStr, err := pgc.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	runMigrations(t, connStr)

	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	return pool
}

func runMigrations(t *testing.T, connStr string) {
	t.Helper()

	m, err := migrate.New("file://../../../migrations", connStr)

	require.NoError(t, err)
	defer m.Close()

	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		require.NoError(t, err)
	}
}

func newTestService(t *testing.T, pool *pgxpool.Pool, verifier google.TokenVerifier) auth.UserAuthService {
	t.Helper()
	authRepo := auth.NewUserAuthRepository(pool)
	userRepo := user.NewUserRepository(pool)
	repRepo := reputation.NewUserReputationRepository(pool)
	repSvc := reputation.NewUserReputationService(repRepo)
	jwtMgr := jwt.NewManager("test-secret-at-least-32-bytes!!", 24*time.Hour)
	return auth.NewUserAuthService(authRepo, userRepo, repSvc, verifier, jwtMgr, pool)
}

func TestGoogleLogin_NewUser_RegistersAndReturnsToken(t *testing.T) {
	pool := startPostgres(t)
	claims := validClaims()
	svc := newTestService(t, pool, &fakeVerifier{claims: claims})

	result, err := svc.GoogleLogin(context.Background(), "any-id-token")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.NotEmpty(t, result.Token)
	assert.Equal(t, claims.Email, result.User.Email)
	assert.Equal(t, claims.Name, result.User.Name)
	assert.NotEqual(t, uuid.Nil, result.User.ID)
}

func TestGoogleLogin_ExistingUser_LoginsWithoutDuplicate(t *testing.T) {
	pool := startPostgres(t)
	claims := validClaims()
	svc := newTestService(t, pool, &fakeVerifier{claims: claims})
	ctx := context.Background()

	first, err := svc.GoogleLogin(ctx, "token")
	require.NoError(t, err)

	second, err := svc.GoogleLogin(ctx, "token")
	require.NoError(t, err)

	assert.Equal(t, first.User.ID, second.User.ID, "must resolve to the same user")
	assert.NotEmpty(t, second.Token)
}

func TestGoogleLogin_ExistingUser_TokenChangesAcrossCalls(t *testing.T) {
	pool := startPostgres(t)
	claims := validClaims()
	authRepo := auth.NewUserAuthRepository(pool)
	userRepo := user.NewUserRepository(pool)
	repRepo := reputation.NewUserReputationRepository(pool)
	repSvc := reputation.NewUserReputationService(repRepo)
	jwtMgr := jwt.NewManager("test-secret-at-least-32-bytes!!", time.Second)
	svc := auth.NewUserAuthService(authRepo, userRepo, repSvc, &fakeVerifier{claims: claims}, jwtMgr, pool)

	ctx := context.Background()
	first, _ := svc.GoogleLogin(ctx, "t")

	time.Sleep(time.Second + 100*time.Millisecond)

	second, err := svc.GoogleLogin(ctx, "t")
	require.NoError(t, err)

	assert.NotEqual(t, first.Token, second.Token, "tokens issued at different times must differ")
}

func TestGoogleLogin_InvalidGoogleToken_ReturnsError(t *testing.T) {
	pool := startPostgres(t)
	svc := newTestService(t, pool, &fakeVerifier{err: errors.New("token signature mismatch")})

	result, err := svc.GoogleLogin(context.Background(), "bad-token")

	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid google token")
}

func TestGoogleLogin_NewUser_ReputationRowCreated(t *testing.T) {
	pool := startPostgres(t)
	claims := validClaims()
	svc := newTestService(t, pool, &fakeVerifier{claims: claims})

	result, err := svc.GoogleLogin(context.Background(), "token")
	require.NoError(t, err)

	var count int
	err = pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM user_reputations WHERE user_id = $1`, result.User.ID,
	).Scan(&count)

	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestGoogleLogin_NewUser_AuthRecordLinkedToUser(t *testing.T) {
	pool := startPostgres(t)
	claims := validClaims()
	svc := newTestService(t, pool, &fakeVerifier{claims: claims})

	result, err := svc.GoogleLogin(context.Background(), "token")
	require.NoError(t, err)

	var googleSub string
	err = pool.QueryRow(context.Background(),
		`SELECT google_sub FROM user_auth WHERE user_id = $1`, result.User.ID,
	).Scan(&googleSub)

	require.NoError(t, err)
	assert.Equal(t, claims.Sub, googleSub)
}

func TestGoogleLogin_DifferentSubs_CreateSeparateUsers(t *testing.T) {
	pool := startPostgres(t)
	ctx := context.Background()

	claimsA := &google.Claims{Sub: "sub-aaa", Email: "a@example.com", Name: "A"}
	claimsB := &google.Claims{Sub: "sub-bbb", Email: "b@example.com", Name: "B"}

	resA, err := newTestService(t, pool, &fakeVerifier{claims: claimsA}).GoogleLogin(ctx, "ta")
	require.NoError(t, err)

	resB, err := newTestService(t, pool, &fakeVerifier{claims: claimsB}).GoogleLogin(ctx, "tb")
	require.NoError(t, err)

	assert.NotEqual(t, resA.User.ID, resB.User.ID)
}

func TestGoogleLogin_Rollback_WhenReputationFails_LeavesNoOrphanedRows(t *testing.T) {

	pool := startPostgres(t)
	claims := validClaims()

	authRepo := auth.NewUserAuthRepository(pool)
	userRepo := user.NewUserRepository(pool)
	repSvc := reputation.NewUserReputationService(&failingReputationRepo{})
	jwtMgr := jwt.NewManager("test-secret-at-least-32-bytes!!", 24*time.Hour)
	svc := auth.NewUserAuthService(authRepo, userRepo, repSvc, &fakeVerifier{claims: claims}, jwtMgr, pool)

	_, err := svc.GoogleLogin(context.Background(), "token")
	require.Error(t, err, "GoogleLogin must fail when reputation init fails")

	var userCount, authCount int
	_ = pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM users WHERE email = $1`, claims.Email,
	).Scan(&userCount)
	_ = pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM user_auth WHERE google_sub = $1`, claims.Sub,
	).Scan(&authCount)

	assert.Equal(t, 0, userCount, "user row must be rolled back")
	assert.Equal(t, 0, authCount, "user_auth row must be rolled back")
}

type failingReputationRepo struct{}

func (f *failingReputationRepo) Save(_ context.Context, _ *reputation.UserReputation) (*reputation.UserReputation, error) {
	return nil, errors.New("reputation store unavailable")
}

func (f *failingReputationRepo) FindByUserID(_ context.Context, _ uuid.UUID) (*reputation.UserReputation, error) {
	return nil, errors.New("reputation store unavailable")
}

func (f *failingReputationRepo) Update(_ context.Context, _ *reputation.UserReputation) (*reputation.UserReputation, error) {
	return nil, errors.New("reputation store unavailable")
}
