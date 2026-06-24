CREATE EXTENSION IF NOT EXISTS postgis;


CREATE TABLE user_locations (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    location   GEOGRAPHY(Point, 4326) NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_flagged BOOL NOT NULL DEFAULT FALSE
);

CREATE INDEX idx_user_locations_location ON user_locations USING GIST(location);
