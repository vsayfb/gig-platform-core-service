//go:build integration

package location_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/vsayfb/gig-platform-core-service/internal/user/location"
)

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

	m, err := migrate.New("file://../../../migrations", connStr)
	require.NoError(t, err)
	defer m.Close()

	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		require.NoError(t, err)
	}
}

func newTestService(t *testing.T, pool *pgxpool.Pool) *location.UserLocationService {
	t.Helper()
	repo := location.NewUserLocationRepository(pool)
	return location.NewUserLocationService(repo)
}

// seedUser inserts a bare user row so FK constraints on user_locations are satisfied.
func seedUser(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO users (id, name, email)
		VALUES ($1, $2, $3)
	`, id, "Test User", id.String()+"@example.com")
	require.NoError(t, err)
	return id
}

func TestUpsert_NewLocation_SavesAndReturns(t *testing.T) {
	pool := startPostgres(t)
	svc := newTestService(t, pool)
	userID := seedUser(t, pool)

	loc, err := svc.Upsert(context.Background(), userID, 41.015137, 28.979530) // Istanbul
	require.NoError(t, err)

	assert.NotEqual(t, uuid.Nil, loc.ID)
	assert.Equal(t, userID, loc.UserID)
	assert.InDelta(t, 41.015137, loc.Lat, 0.0001)
	assert.InDelta(t, 28.979530, loc.Lng, 0.0001)
	assert.False(t, loc.IsFlagged)
}

func TestUpsert_NewLocation_PersistedToDatabase(t *testing.T) {
	pool := startPostgres(t)
	svc := newTestService(t, pool)
	userID := seedUser(t, pool)

	_, err := svc.Upsert(context.Background(), userID, 41.015137, 28.979530)
	require.NoError(t, err)

	var count int
	err = pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM user_locations WHERE user_id = $1`, userID,
	).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestUpsert_ExistingLocation_UpdatesCoordinates(t *testing.T) {
	pool := startPostgres(t)
	svc := newTestService(t, pool)
	userID := seedUser(t, pool)
	ctx := context.Background()

	_, err := svc.Upsert(ctx, userID, 41.015137, 28.979530) // Istanbul
	require.NoError(t, err)

	updated, err := svc.Upsert(ctx, userID, 41.015137, 28.979530)
	require.NoError(t, err)

	assert.InDelta(t, 41.015137, updated.Lat, 0.0001)
	assert.InDelta(t, 28.979530, updated.Lng, 0.0001)
}

func TestUpsert_ExistingLocation_OnlyOneRowInDatabase(t *testing.T) {
	pool := startPostgres(t)
	svc := newTestService(t, pool)
	userID := seedUser(t, pool)
	ctx := context.Background()

	_, err := svc.Upsert(ctx, userID, 41.015137, 28.979530)
	require.NoError(t, err)
	_, err = svc.Upsert(ctx, userID, 39.925533, 32.866287) // Ankara
	require.NoError(t, err)

	var count int
	err = pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM user_locations WHERE user_id = $1`, userID,
	).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "upsert must not create duplicate rows")
}

func TestUpsert_InvalidLatitude_ReturnsError(t *testing.T) {
	pool := startPostgres(t)
	svc := newTestService(t, pool)
	userID := seedUser(t, pool)

	_, err := svc.Upsert(context.Background(), userID, 91.0, 28.979530)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid latitude")
}

func TestUpsert_InvalidLongitude_ReturnsError(t *testing.T) {
	pool := startPostgres(t)
	svc := newTestService(t, pool)
	userID := seedUser(t, pool)

	_, err := svc.Upsert(context.Background(), userID, 41.015137, 181.0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid longitude")
}

func TestUpsert_BoundaryCoordinates_AreValid(t *testing.T) {
	pool := startPostgres(t)
	svc := newTestService(t, pool)
	ctx := context.Background()

	cases := []struct {
		name string
		lat  float64
		lng  float64
	}{
		{"north pole", 90.0, 0.0},
		{"south pole", -90.0, 0.0},
		{"date line east", 0.0, 180.0},
		{"date line west", 0.0, -180.0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			userID := seedUser(t, pool)
			_, err := svc.Upsert(ctx, userID, tc.lat, tc.lng)
			assert.NoError(t, err, "boundary coordinate must be valid")
		})
	}
}

func TestUpsert_SuspiciousVelocity_FlagsLocation(t *testing.T) {
	// Simulate a user jumping from Istanbul to New York (~8500 km) in under
	// a second — well above the 200 km/h threshold.
	pool := startPostgres(t)
	svc := newTestService(t, pool)
	userID := seedUser(t, pool)
	ctx := context.Background()

	_, err := svc.Upsert(ctx, userID, 41.015137, 28.979530) // Istanbul
	require.NoError(t, err)

	// Force updated_at to be very recent so elapsed time ≈ 0
	_, err = pool.Exec(ctx,
		`UPDATE user_locations SET updated_at = NOW() WHERE user_id = $1`, userID,
	)
	require.NoError(t, err)

	result, err := svc.Upsert(ctx, userID, 40.712776, -74.005974) // New York
	require.NoError(t, err)

	assert.True(t, result.IsFlagged, "teleporting user must be flagged")
}

func TestUpsert_NormalVelocity_DoesNotFlag(t *testing.T) {
	pool := startPostgres(t)
	svc := newTestService(t, pool)
	userID := seedUser(t, pool)
	ctx := context.Background()

	_, err := svc.Upsert(ctx, userID, 41.015137, 28.979530) // Istanbul
	require.NoError(t, err)

	// Simulate 2 hours passing — user moves ~10 km (well within 200 km/h)
	_, err = pool.Exec(ctx,
		`UPDATE user_locations SET updated_at = NOW() - INTERVAL '2 hours' WHERE user_id = $1`, userID,
	)
	require.NoError(t, err)

	result, err := svc.Upsert(ctx, userID, 41.105137, 28.979530) // ~10 km north
	require.NoError(t, err)

	assert.False(t, result.IsFlagged, "reasonable movement must not be flagged")
}

func TestFindNearby_ReturnsUsersWithinRadius(t *testing.T) {
	pool := startPostgres(t)
	svc := newTestService(t, pool)
	ctx := context.Background()

	requester := seedUser(t, pool)

	// Place two users near Istanbul
	nearbyA := seedUser(t, pool)
	nearbyB := seedUser(t, pool)
	far := seedUser(t, pool)

	_, err := svc.Upsert(ctx, nearbyA, 41.015137, 28.979530) // Istanbul centre
	require.NoError(t, err)
	_, err = svc.Upsert(ctx, nearbyB, 41.020000, 28.985000) // ~700m away
	require.NoError(t, err)
	_, err = svc.Upsert(ctx, far, 39.925533, 32.866287) // Ankara ~350 km away
	require.NoError(t, err)

	results, err := svc.FindNearby(ctx, requester, 41.015137, 28.979530, 5.0) // 5 km radius
	require.NoError(t, err)

	ids := make([]uuid.UUID, len(results))
	for i, r := range results {
		ids[i] = r.UserID
	}

	assert.Contains(t, ids, nearbyA)
	assert.Contains(t, ids, nearbyB)
	assert.NotContains(t, ids, far, "Ankara must be outside 5 km radius")
}

func TestFindNearby_ExcludesRequester(t *testing.T) {
	pool := startPostgres(t)
	svc := newTestService(t, pool)
	ctx := context.Background()

	requester := seedUser(t, pool)
	_, err := svc.Upsert(ctx, requester, 41.015137, 28.979530)
	require.NoError(t, err)

	results, err := svc.FindNearby(ctx, requester, 41.015137, 28.979530, 10.0)
	require.NoError(t, err)

	for _, r := range results {
		assert.NotEqual(t, requester, r.UserID, "requester must be excluded from results")
	}
}

func TestFindNearby_EmptyRadius_ReturnsError(t *testing.T) {
	pool := startPostgres(t)
	svc := newTestService(t, pool)
	userID := seedUser(t, pool)

	_, err := svc.FindNearby(context.Background(), userID, 41.015137, 28.979530, 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "radiusKm must be positive")
}

func TestFindNearby_NoUsersNearby_ReturnsEmptySlice(t *testing.T) {
	pool := startPostgres(t)
	svc := newTestService(t, pool)
	ctx := context.Background()

	requester := seedUser(t, pool)
	other := seedUser(t, pool)

	// Place other user in Ankara, search from Istanbul
	_, err := svc.Upsert(ctx, other, 39.925533, 32.866287)
	require.NoError(t, err)

	results, err := svc.FindNearby(ctx, requester, 41.015137, 28.979530, 5.0)
	require.NoError(t, err)
	assert.Empty(t, results)
}
