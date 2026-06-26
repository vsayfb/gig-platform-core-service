//go:build integration

package gig_test

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
		id, "Test User", id.String()+"@example.com",
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

// ---------------------------------------------------------------------------
// Factory helpers
// ---------------------------------------------------------------------------

func newTestService(pool *pgxpool.Pool) *gig.GigService {
	return gig.NewGigService(gig.NewRepository(pool))
}

func validCreateInput(lat, lng float64, categoryIDs ...uuid.UUID) gig.CreateGigInput {
	start := time.Now().UTC().Add(24 * time.Hour).Truncate(24 * time.Hour)
	end := start.Add(7 * 24 * time.Hour)
	expires := start.Add(48 * time.Hour)
	return gig.CreateGigInput{
		Title:            "Need a plumber",
		DescriptionRaw:   "Fix the kitchen sink",
		DescriptionClean: "Fix the kitchen sink",
		DurationType:     gig.DurationDaily,
		StartDate:        start,
		EndDate:          &end,
		Slots:            2,
		ExpiresAt:        &expires,
		Lat:              lat,
		Lng:              lng,
		City:             "Istanbul",
		District:         "Kadikoy",
		CategoryIDs:      categoryIDs,
	}
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestCreate_ValidInput_ReturnsGigDetail(t *testing.T) {
	pool := startPostgres(t)
	svc := newTestService(pool)
	posterID := seedUser(t, pool)
	catID := seedCategory(t, pool)

	detail, err := svc.Create(context.Background(), posterID, validCreateInput(41.01, 28.97, catID))
	require.NoError(t, err)

	assert.NotEqual(t, uuid.Nil, detail.Gig.ID)
	assert.Equal(t, posterID, detail.Gig.PosterID)
	assert.Equal(t, "Need a plumber", detail.Gig.Title)
	assert.Equal(t, gig.StatusOpen, detail.Gig.Status)
	assert.Equal(t, gig.DurationDaily, detail.Gig.DurationType)
	assert.Equal(t, 2, detail.Gig.Slots)
	require.NotNil(t, detail.Location)
	assert.InDelta(t, 41.01, detail.Location.Lat, 0.0001)
	assert.InDelta(t, 28.97, detail.Location.Lng, 0.0001)
	assert.Equal(t, "Istanbul", detail.Location.City)
	assert.Equal(t, []uuid.UUID{catID}, detail.Categories)
}

func TestCreate_MissingTitle_ReturnsError(t *testing.T) {
	pool := startPostgres(t)
	svc := newTestService(pool)
	posterID := seedUser(t, pool)

	in := validCreateInput(41.01, 28.97)
	in.Title = ""

	_, err := svc.Create(context.Background(), posterID, in)
	require.ErrorIs(t, err, gig.ErrInvalidInput)
}

func TestCreate_MissingDescription_ReturnsError(t *testing.T) {
	pool := startPostgres(t)
	svc := newTestService(pool)
	posterID := seedUser(t, pool)

	in := validCreateInput(41.01, 28.97)
	in.DescriptionRaw = ""

	_, err := svc.Create(context.Background(), posterID, in)
	require.ErrorIs(t, err, gig.ErrInvalidInput)
}

func TestCreate_InvalidDurationType_ReturnsError(t *testing.T) {
	pool := startPostgres(t)
	svc := newTestService(pool)
	posterID := seedUser(t, pool)

	in := validCreateInput(41.01, 28.97)
	in.DurationType = "HOURLY"

	_, err := svc.Create(context.Background(), posterID, in)
	require.ErrorIs(t, err, gig.ErrInvalidInput)
}

func TestCreate_ZeroCoordinates_ReturnsError(t *testing.T) {
	pool := startPostgres(t)
	svc := newTestService(pool)
	posterID := seedUser(t, pool)

	in := validCreateInput(0, 0)

	_, err := svc.Create(context.Background(), posterID, in)
	require.ErrorIs(t, err, gig.ErrInvalidInput)
}

func TestCreate_NoCategoryIDs_CreatesGigWithEmptyCategories(t *testing.T) {
	pool := startPostgres(t)
	svc := newTestService(pool)
	posterID := seedUser(t, pool)

	detail, err := svc.Create(context.Background(), posterID, validCreateInput(41.01, 28.97))
	require.NoError(t, err)
	assert.Empty(t, detail.Categories)
}

func TestCreate_SlotsBelowOne_DefaultsToOne(t *testing.T) {
	pool := startPostgres(t)
	svc := newTestService(pool)
	posterID := seedUser(t, pool)

	in := validCreateInput(41.01, 28.97)
	in.Slots = 0

	detail, err := svc.Create(context.Background(), posterID, in)
	require.NoError(t, err)
	assert.Equal(t, 1, detail.Gig.Slots)
}

// ---------------------------------------------------------------------------
// Get
// ---------------------------------------------------------------------------

func TestGet_ExistingGig_ReturnsDetail(t *testing.T) {
	pool := startPostgres(t)
	svc := newTestService(pool)
	posterID := seedUser(t, pool)

	created, err := svc.Create(context.Background(), posterID, validCreateInput(41.01, 28.97))
	require.NoError(t, err)

	fetched, err := svc.Get(context.Background(), created.Gig.ID)
	require.NoError(t, err)

	assert.Equal(t, created.Gig.ID, fetched.Gig.ID)
	assert.Equal(t, created.Gig.Title, fetched.Gig.Title)
}

func TestGet_NonExistentGig_ReturnsErrGigNotFound(t *testing.T) {
	pool := startPostgres(t)
	svc := newTestService(pool)

	_, err := svc.Get(context.Background(), uuid.New())
	require.ErrorIs(t, err, gig.ErrGigNotFound)
}

// ---------------------------------------------------------------------------
// Edit
// ---------------------------------------------------------------------------

func TestEdit_ValidInput_UpdatesTitle(t *testing.T) {
	pool := startPostgres(t)
	svc := newTestService(pool)
	posterID := seedUser(t, pool)

	created, err := svc.Create(context.Background(), posterID, validCreateInput(41.01, 28.97))
	require.NoError(t, err)

	newTitle := "Updated title"
	updated, err := svc.Edit(context.Background(), created.Gig.ID, posterID, gig.UpdateGigInput{
		Title: &newTitle,
	})
	require.NoError(t, err)
	assert.Equal(t, newTitle, updated.Gig.Title)
}

func TestEdit_NotPoster_ReturnsErrNotPoster(t *testing.T) {
	pool := startPostgres(t)
	svc := newTestService(pool)
	posterID := seedUser(t, pool)
	otherID := seedUser(t, pool)

	created, err := svc.Create(context.Background(), posterID, validCreateInput(41.01, 28.97))
	require.NoError(t, err)

	newTitle := "Hijacked title"
	_, err = svc.Edit(context.Background(), created.Gig.ID, otherID, gig.UpdateGigInput{
		Title: &newTitle,
	})
	require.ErrorIs(t, err, gig.ErrNotPoster)
}

func TestEdit_NonOpenGig_ReturnsErrGigNotEditable(t *testing.T) {
	pool := startPostgres(t)
	svc := newTestService(pool)
	posterID := seedUser(t, pool)
	ctx := context.Background()

	created, err := svc.Create(ctx, posterID, validCreateInput(41.01, 28.97))
	require.NoError(t, err)

	// Cancel it so status is no longer OPEN.
	err = svc.Cancel(ctx, created.Gig.ID, posterID)
	require.NoError(t, err)

	newTitle := "Should not work"
	_, err = svc.Edit(ctx, created.Gig.ID, posterID, gig.UpdateGigInput{Title: &newTitle})
	require.ErrorIs(t, err, gig.ErrGigNotEditable)
}

func TestEdit_NonExistentGig_ReturnsErrGigNotFound(t *testing.T) {
	pool := startPostgres(t)
	svc := newTestService(pool)
	posterID := seedUser(t, pool)

	newTitle := "Ghost gig"
	_, err := svc.Edit(context.Background(), uuid.New(), posterID, gig.UpdateGigInput{Title: &newTitle})
	require.ErrorIs(t, err, gig.ErrGigNotFound)
}

// ---------------------------------------------------------------------------
// Cancel
// ---------------------------------------------------------------------------

func TestCancel_OpenGig_Succeeds(t *testing.T) {
	pool := startPostgres(t)
	svc := newTestService(pool)
	posterID := seedUser(t, pool)
	ctx := context.Background()

	created, err := svc.Create(ctx, posterID, validCreateInput(41.01, 28.97))
	require.NoError(t, err)

	err = svc.Cancel(ctx, created.Gig.ID, posterID)
	require.NoError(t, err)

	fetched, err := svc.Get(ctx, created.Gig.ID)
	require.NoError(t, err)
	assert.Equal(t, gig.StatusCancelled, fetched.Gig.Status)
}

func TestCancel_NotPoster_ReturnsErrNotPoster(t *testing.T) {
	pool := startPostgres(t)
	svc := newTestService(pool)
	posterID := seedUser(t, pool)
	otherID := seedUser(t, pool)
	ctx := context.Background()

	created, err := svc.Create(ctx, posterID, validCreateInput(41.01, 28.97))
	require.NoError(t, err)

	err = svc.Cancel(ctx, created.Gig.ID, otherID)
	require.ErrorIs(t, err, gig.ErrNotPoster)
}

func TestCancel_CompletedGig_ReturnsErrGigNotCancellable(t *testing.T) {
	pool := startPostgres(t)
	svc := newTestService(pool)
	posterID := seedUser(t, pool)
	ctx := context.Background()

	created, err := svc.Create(ctx, posterID, validCreateInput(41.01, 28.97))
	require.NoError(t, err)

	// Force status to COMPLETED directly in DB.
	_, err = pool.Exec(ctx,
		`UPDATE gigs SET status = 'COMPLETED' WHERE id = $1`, created.Gig.ID,
	)
	require.NoError(t, err)

	err = svc.Cancel(ctx, created.Gig.ID, posterID)
	require.ErrorIs(t, err, gig.ErrGigNotCancellable)
}

func TestCancel_NonExistentGig_ReturnsErrGigNotFound(t *testing.T) {
	pool := startPostgres(t)
	svc := newTestService(pool)
	posterID := seedUser(t, pool)

	err := svc.Cancel(context.Background(), uuid.New(), posterID)
	require.ErrorIs(t, err, gig.ErrGigNotFound)
}

// ---------------------------------------------------------------------------
// Feed
// ---------------------------------------------------------------------------

func TestFeed_ReturnsGigsWithinRadius(t *testing.T) {
	pool := startPostgres(t)
	svc := newTestService(pool)
	ctx := context.Background()

	posterID := seedUser(t, pool)

	// Istanbul — within 5 km of search point
	near, err := svc.Create(ctx, posterID, validCreateInput(41.015137, 28.979530))
	require.NoError(t, err)

	// Ankara — ~350 km away, outside any reasonable radius
	_, err = svc.Create(ctx, posterID, validCreateInput(39.925533, 32.866287))
	require.NoError(t, err)

	results, err := svc.Feed(ctx, gig.FeedParams{
		Lat:          41.015137,
		Lng:          28.979530,
		RadiusMeters: 5000,
	})
	require.NoError(t, err)

	ids := make([]uuid.UUID, len(results))
	for i, r := range results {
		ids[i] = r.Gig.ID
	}
	assert.Contains(t, ids, near.Gig.ID)
	assert.Len(t, results, 1, "only Istanbul gig should be within 5 km")
}

func TestFeed_FilterByDurationType(t *testing.T) {
	pool := startPostgres(t)
	svc := newTestService(pool)
	ctx := context.Background()
	posterID := seedUser(t, pool)

	daily, err := svc.Create(ctx, posterID, validCreateInput(41.01, 28.97))
	require.NoError(t, err)

	weeklyIn := validCreateInput(41.01, 28.97)
	weeklyIn.DurationType = gig.DurationWeekly
	_, err = svc.Create(ctx, posterID, weeklyIn)
	require.NoError(t, err)

	dt := gig.DurationDaily
	results, err := svc.Feed(ctx, gig.FeedParams{
		Lat:          41.01,
		Lng:          28.97,
		RadiusMeters: 5000,
		DurationType: &dt,
	})
	require.NoError(t, err)

	for _, r := range results {
		assert.Equal(t, gig.DurationDaily, r.Gig.DurationType)
	}
	ids := make([]uuid.UUID, len(results))
	for i, r := range results {
		ids[i] = r.Gig.ID
	}
	assert.Contains(t, ids, daily.Gig.ID)
}

func TestFeed_FilterByCategory(t *testing.T) {
	pool := startPostgres(t)
	svc := newTestService(pool)
	ctx := context.Background()
	posterID := seedUser(t, pool)
	catA := seedCategory(t, pool)
	catB := seedCategory(t, pool)

	withA, err := svc.Create(ctx, posterID, validCreateInput(41.01, 28.97, catA))
	require.NoError(t, err)
	_, err = svc.Create(ctx, posterID, validCreateInput(41.01, 28.97, catB))
	require.NoError(t, err)

	results, err := svc.Feed(ctx, gig.FeedParams{
		Lat:          41.01,
		Lng:          28.97,
		RadiusMeters: 5000,
		CategoryID:   &catA,
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, withA.Gig.ID, results[0].Gig.ID)
}

func TestFeed_CancelledGigsNotReturned(t *testing.T) {
	pool := startPostgres(t)
	svc := newTestService(pool)
	ctx := context.Background()
	posterID := seedUser(t, pool)

	created, err := svc.Create(ctx, posterID, validCreateInput(41.01, 28.97))
	require.NoError(t, err)

	err = svc.Cancel(ctx, created.Gig.ID, posterID)
	require.NoError(t, err)

	results, err := svc.Feed(ctx, gig.FeedParams{
		Lat:          41.01,
		Lng:          28.97,
		RadiusMeters: 5000,
	})
	require.NoError(t, err)

	for _, r := range results {
		assert.NotEqual(t, created.Gig.ID, r.Gig.ID, "cancelled gig must not appear in feed")
	}
}

func TestFeed_LimitDefaultsTo20_WhenZero(t *testing.T) {
	pool := startPostgres(t)
	svc := newTestService(pool)
	ctx := context.Background()
	posterID := seedUser(t, pool)

	// Create 25 gigs, all at the same location
	for i := 0; i < 25; i++ {
		_, err := svc.Create(ctx, posterID, validCreateInput(41.01, 28.97))
		require.NoError(t, err)
	}

	results, err := svc.Feed(ctx, gig.FeedParams{
		Lat:          41.01,
		Lng:          28.97,
		RadiusMeters: 5000,
		Limit:        0, // should default to 20
	})
	require.NoError(t, err)
	assert.LessOrEqual(t, len(results), 20, "feed must not exceed default limit of 20")
}

func TestFeed_KeysetPagination_CursorExcludesOlderGigs(t *testing.T) {
	pool := startPostgres(t)
	svc := newTestService(pool)
	ctx := context.Background()
	posterID := seedUser(t, pool)

	first, err := svc.Create(ctx, posterID, validCreateInput(41.01, 28.97))
	require.NoError(t, err)

	// Small sleep so created_at differs
	time.Sleep(10 * time.Millisecond)

	second, err := svc.Create(ctx, posterID, validCreateInput(41.01, 28.97))
	require.NoError(t, err)

	// Use second gig's created_at as cursor — should only return first
	cursor := second.Gig.CreatedAt
	results, err := svc.Feed(ctx, gig.FeedParams{
		Lat:          41.01,
		Lng:          28.97,
		RadiusMeters: 5000,
		Cursor:       &cursor,
	})
	require.NoError(t, err)

	ids := make([]uuid.UUID, len(results))
	for i, r := range results {
		ids[i] = r.Gig.ID
	}
	assert.Contains(t, ids, first.Gig.ID)
	assert.NotContains(t, ids, second.Gig.ID, "gig at cursor boundary must be excluded")
}
