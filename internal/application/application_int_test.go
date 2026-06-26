//go:build integration

package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/vsayfb/gig-platform-core-service/internal/application"
	"github.com/vsayfb/gig-platform-core-service/internal/gig"
)

// ---------------------------------------------------------------------------
// Test DB
// ---------------------------------------------------------------------------

func startPostgres(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()

	pgc, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgis/postgis:16-3.4"),
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
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
	m, err := migrate.New("file://../../migrations", connStr)
	require.NoError(t, err)
	defer m.Close()
	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		require.NoError(t, err)
	}
}

// ---------------------------------------------------------------------------
// Seed helpers
// ---------------------------------------------------------------------------

func seedUser(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO users (id, name, email) VALUES ($1, $2, $3)`,
		id, "User "+id.String(), id.String()+"@example.com",
	)
	require.NoError(t, err)
	return id
}

func seedCategory(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO categories (id, name, slug, status) VALUES ($1, $2, $3, 'ACTIVE')`,
		id, "cat-"+id.String(), "slug-"+id.String(),
	)
	require.NoError(t, err)
	return id
}

func seedGig(t *testing.T, pool *pgxpool.Pool, posterID uuid.UUID) uuid.UUID {
	t.Helper()
	gigSvc := gig.NewGigService(gig.NewRepository(pool))
	start := time.Now().UTC().Add(24 * time.Hour).Truncate(24 * time.Hour)
	end := start.Add(7 * 24 * time.Hour)
	expires := start.Add(48 * time.Hour)

	detail, err := gigSvc.Create(context.Background(), posterID, gig.CreateGigInput{
		Title:            "Test Gig",
		DescriptionRaw:   "Description",
		DescriptionClean: "Description",
		DurationType:     gig.DurationDaily,
		StartDate:        start,
		EndDate:          &end,
		Slots:            2,
		ExpiresAt:        &expires,
		Lat:              41.01,
		Lng:              28.97,
		City:             "Istanbul",
		District:         "Kadikoy",
	})
	require.NoError(t, err)
	return detail.Gig.ID
}

// ---------------------------------------------------------------------------
// Service factory
// ---------------------------------------------------------------------------

func newTestService(pool *pgxpool.Pool) application.ApplicationService {
	appRepo := application.NewRepository(pool)
	gigRepo := gig.NewRepository(pool)
	return application.NewApplicationService(appRepo, gigRepo)
}

// ---------------------------------------------------------------------------
// Apply
// ---------------------------------------------------------------------------

func TestApply_ValidRequest_ReturnsApplication(t *testing.T) {
	pool := startPostgres(t)
	svc := newTestService(pool)
	ctx := context.Background()

	posterID := seedUser(t, pool)
	applicantID := seedUser(t, pool)
	gigID := seedGig(t, pool, posterID)

	a, err := svc.Apply(ctx, gigID, applicantID)
	require.NoError(t, err)

	assert.NotEqual(t, uuid.Nil, a.ID)
	assert.Equal(t, gigID, a.GigID)
	assert.Equal(t, applicantID, a.ApplicantID)
	assert.Equal(t, application.StatusPending, a.Status)
}

func TestApply_GigNotFound_ReturnsError(t *testing.T) {
	pool := startPostgres(t)
	svc := newTestService(pool)
	applicantID := seedUser(t, pool)

	_, err := svc.Apply(context.Background(), uuid.New(), applicantID)
	require.ErrorIs(t, err, gig.ErrGigNotFound)
}

func TestApply_CannotApplyOwnGig_ReturnsError(t *testing.T) {
	pool := startPostgres(t)
	svc := newTestService(pool)
	ctx := context.Background()

	posterID := seedUser(t, pool)
	gigID := seedGig(t, pool, posterID)

	_, err := svc.Apply(ctx, gigID, posterID)
	require.ErrorIs(t, err, application.ErrCannotApplyOwn)
}

func TestApply_DuplicateApplication_ReturnsError(t *testing.T) {
	pool := startPostgres(t)
	svc := newTestService(pool)
	ctx := context.Background()

	posterID := seedUser(t, pool)
	applicantID := seedUser(t, pool)
	gigID := seedGig(t, pool, posterID)

	_, err := svc.Apply(ctx, gigID, applicantID)
	require.NoError(t, err)

	_, err = svc.Apply(ctx, gigID, applicantID)
	require.ErrorIs(t, err, application.ErrAlreadyApplied)
}

func TestApply_GigNotOpen_ReturnsError(t *testing.T) {
	pool := startPostgres(t)
	svc := newTestService(pool)
	ctx := context.Background()

	posterID := seedUser(t, pool)
	applicantID := seedUser(t, pool)
	gigID := seedGig(t, pool, posterID)

	// Cancel the gig so it's no longer OPEN.
	gigSvc := gig.NewGigService(gig.NewRepository(pool))
	err := gigSvc.Cancel(ctx, gigID, posterID)
	require.NoError(t, err)

	_, err = svc.Apply(ctx, gigID, applicantID)
	require.ErrorIs(t, err, application.ErrGigNotOpen)
}

func TestApply_MultipleApplicants_AllPending(t *testing.T) {
	pool := startPostgres(t)
	svc := newTestService(pool)
	ctx := context.Background()

	posterID := seedUser(t, pool)
	gigID := seedGig(t, pool, posterID)

	for i := 0; i < 3; i++ {
		applicantID := seedUser(t, pool)
		a, err := svc.Apply(ctx, gigID, applicantID)
		require.NoError(t, err)
		assert.Equal(t, application.StatusPending, a.Status)
	}
}

// ---------------------------------------------------------------------------
// Get
// ---------------------------------------------------------------------------

func TestGet_AsApplicant_ReturnsApplication(t *testing.T) {
	pool := startPostgres(t)
	svc := newTestService(pool)
	ctx := context.Background()

	posterID := seedUser(t, pool)
	applicantID := seedUser(t, pool)
	gigID := seedGig(t, pool, posterID)

	created, err := svc.Apply(ctx, gigID, applicantID)
	require.NoError(t, err)

	fetched, err := svc.Get(ctx, created.ID, applicantID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, fetched.ID)
}

func TestGet_AsPoster_ReturnsApplication(t *testing.T) {
	pool := startPostgres(t)
	svc := newTestService(pool)
	ctx := context.Background()

	posterID := seedUser(t, pool)
	applicantID := seedUser(t, pool)
	gigID := seedGig(t, pool, posterID)

	created, err := svc.Apply(ctx, gigID, applicantID)
	require.NoError(t, err)

	// Poster can also view the application.
	fetched, err := svc.Get(ctx, created.ID, posterID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, fetched.ID)
}

func TestGet_AsUnrelatedUser_ReturnsForbidden(t *testing.T) {
	pool := startPostgres(t)
	svc := newTestService(pool)
	ctx := context.Background()

	posterID := seedUser(t, pool)
	applicantID := seedUser(t, pool)
	randomID := seedUser(t, pool)
	gigID := seedGig(t, pool, posterID)

	created, err := svc.Apply(ctx, gigID, applicantID)
	require.NoError(t, err)

	_, err = svc.Get(ctx, created.ID, randomID)
	require.ErrorIs(t, err, application.ErrNotApplicant)
}

func TestGet_NotFound_ReturnsError(t *testing.T) {
	pool := startPostgres(t)
	svc := newTestService(pool)
	userID := seedUser(t, pool)

	_, err := svc.Get(context.Background(), uuid.New(), userID)
	require.ErrorIs(t, err, application.ErrNotFound)
}

// ---------------------------------------------------------------------------
// ListByGig
// ---------------------------------------------------------------------------

func TestListByGig_AsPoster_ReturnsAllApplications(t *testing.T) {
	pool := startPostgres(t)
	svc := newTestService(pool)
	ctx := context.Background()

	posterID := seedUser(t, pool)
	gigID := seedGig(t, pool, posterID)

	for i := 0; i < 3; i++ {
		applicantID := seedUser(t, pool)
		_, err := svc.Apply(ctx, gigID, applicantID)
		require.NoError(t, err)
	}

	apps, err := svc.ListByGig(ctx, gigID, posterID)
	require.NoError(t, err)
	assert.Len(t, apps, 3)
}

func TestListByGig_AsNonPoster_ReturnsForbidden(t *testing.T) {
	pool := startPostgres(t)
	svc := newTestService(pool)
	ctx := context.Background()

	posterID := seedUser(t, pool)
	otherID := seedUser(t, pool)
	gigID := seedGig(t, pool, posterID)

	_, err := svc.ListByGig(ctx, gigID, otherID)
	require.ErrorIs(t, err, gig.ErrNotPoster)
}

func TestListByGig_GigNotFound_ReturnsError(t *testing.T) {
	pool := startPostgres(t)
	svc := newTestService(pool)
	callerID := seedUser(t, pool)

	_, err := svc.ListByGig(context.Background(), uuid.New(), callerID)
	require.ErrorIs(t, err, gig.ErrGigNotFound)
}

func TestListByGig_NoApplications_ReturnsNil(t *testing.T) {
	pool := startPostgres(t)
	svc := newTestService(pool)
	ctx := context.Background()

	posterID := seedUser(t, pool)
	gigID := seedGig(t, pool, posterID)

	apps, err := svc.ListByGig(ctx, gigID, posterID)
	require.NoError(t, err)
	assert.Empty(t, apps)
}

// ---------------------------------------------------------------------------
// Withdraw
// ---------------------------------------------------------------------------

func TestWithdraw_PendingApplication_Succeeds(t *testing.T) {
	pool := startPostgres(t)
	svc := newTestService(pool)
	ctx := context.Background()

	posterID := seedUser(t, pool)
	applicantID := seedUser(t, pool)
	gigID := seedGig(t, pool, posterID)

	a, err := svc.Apply(ctx, gigID, applicantID)
	require.NoError(t, err)

	err = svc.Withdraw(ctx, a.ID, applicantID)
	require.NoError(t, err)

	// Verify status in DB.
	var status string
	err = pool.QueryRow(ctx,
		`SELECT status FROM applications WHERE id = $1`, a.ID,
	).Scan(&status)
	require.NoError(t, err)
	assert.Equal(t, string(application.StatusWithdrawn), status)
}

func TestWithdraw_NotApplicant_ReturnsForbidden(t *testing.T) {
	pool := startPostgres(t)
	svc := newTestService(pool)
	ctx := context.Background()

	posterID := seedUser(t, pool)
	applicantID := seedUser(t, pool)
	otherID := seedUser(t, pool)
	gigID := seedGig(t, pool, posterID)

	a, err := svc.Apply(ctx, gigID, applicantID)
	require.NoError(t, err)

	err = svc.Withdraw(ctx, a.ID, otherID)
	require.ErrorIs(t, err, application.ErrNotApplicant)
}

func TestWithdraw_NonPendingApplication_ReturnsError(t *testing.T) {
	pool := startPostgres(t)
	svc := newTestService(pool)
	ctx := context.Background()

	posterID := seedUser(t, pool)
	applicantID := seedUser(t, pool)
	gigID := seedGig(t, pool, posterID)

	a, err := svc.Apply(ctx, gigID, applicantID)
	require.NoError(t, err)

	// Force status to HIRED directly in DB.
	_, err = pool.Exec(ctx,
		`UPDATE applications SET status = 'HIRED' WHERE id = $1`, a.ID,
	)
	require.NoError(t, err)

	err = svc.Withdraw(ctx, a.ID, applicantID)
	require.ErrorIs(t, err, application.ErrNotWithdrawable)
}

func TestWithdraw_NotFound_ReturnsError(t *testing.T) {
	pool := startPostgres(t)
	svc := newTestService(pool)
	applicantID := seedUser(t, pool)

	err := svc.Withdraw(context.Background(), uuid.New(), applicantID)
	require.ErrorIs(t, err, application.ErrNotFound)
}
