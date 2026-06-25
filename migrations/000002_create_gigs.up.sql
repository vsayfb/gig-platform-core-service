CREATE TYPE gig_duration_type AS ENUM ('DAILY', 'WEEKLY', 'MONTHLY');
CREATE TYPE gig_status AS ENUM ('OPEN', 'IN_PROGRESS', 'COMPLETED', 'CANCELLED');

CREATE TABLE gigs (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    poster_id           UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title               VARCHAR(255) NOT NULL,
    description_raw     TEXT NOT NULL,
    description_clean   TEXT NOT NULL,
    duration_type       gig_duration_type NOT NULL,
    start_date          DATE NOT NULL,
    end_date            DATE NOT NULL,
    slots               INT NOT NULL DEFAULT 1,
    status              gig_status NOT NULL DEFAULT 'OPEN',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at          TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_gigs_status_created_at ON gigs(status, created_at DESC);
CREATE INDEX idx_gigs_poster_id ON gigs(poster_id);

CREATE TABLE gig_locations (
    id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    gig_id   UUID NOT NULL UNIQUE REFERENCES gigs(id) ON DELETE CASCADE,
    location GEOGRAPHY(Point, 4326) NOT NULL,
    city     VARCHAR(100) NOT NULL,
    district VARCHAR(100) NOT NULL
);

CREATE INDEX idx_gig_locations_location ON gig_locations USING GIST(location);

CREATE TABLE gig_categories (
    gig_id      UUID NOT NULL REFERENCES gigs(id) ON DELETE CASCADE,
    category_id UUID NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    PRIMARY KEY (gig_id, category_id)
)
