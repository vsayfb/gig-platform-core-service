-- 000005_create_gigs.up.sql

CREATE TYPE gig_status AS ENUM ('OPEN', 'IN_PROGRESS', 'COMPLETED', 'CANCELLED');
CREATE TYPE duration_type AS ENUM ('DAILY', 'WEEKLY', 'MONTHLY');

CREATE TABLE gigs (
    id                UUID PRIMARY KEY,
    poster_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title             TEXT NOT NULL,
    description_raw   TEXT NOT NULL,
    description_clean TEXT NOT NULL,
    status            gig_status NOT NULL DEFAULT 'OPEN',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE gig_details (
    gig_id       UUID PRIMARY KEY REFERENCES gigs(id) ON DELETE CASCADE,
    duration_type duration_type,
    start_date   DATE,
    end_date     DATE,
    pay_amount   NUMERIC(12, 2),
    pay_currency CHAR(3),
    expires_at   TIMESTAMPTZ
);

CREATE TABLE gig_locations (
    id       UUID PRIMARY KEY,
    gig_id   UUID NOT NULL UNIQUE REFERENCES gigs(id) ON DELETE CASCADE,
    location GEOGRAPHY(Point, 4326) NOT NULL,
    city     TEXT,
);

CREATE INDEX idx_gig_locations_geoindex ON gig_locations USING GIST (location);

CREATE TABLE gig_categories (
    gig_id      UUID NOT NULL REFERENCES gigs(id) ON DELETE CASCADE,
    category_id UUID NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    PRIMARY KEY (gig_id, category_id)
);

CREATE INDEX idx_gigs_poster_id ON gigs(poster_id);
CREATE INDEX idx_gigs_status ON gigs(status);
CREATE INDEX idx_gigs_created_at ON gigs(created_at DESC);