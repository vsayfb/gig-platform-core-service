CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE users (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name             VARCHAR(100) NOT NULL,
    avatar_url       TEXT,
    email            TEXT NOT NULL UNIQUE,
    bio              TEXT,
    is_verified      BOOL NOT NULL DEFAULT FALSE,
    is_available_today BOOL NOT NULL DEFAULT FALSE,
    last_active_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE user_auth (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    google_sub       VARCHAR(255) NOT NULL UNIQUE,
    phone_encrypted  TEXT,
    phone_hmac       VARCHAR(255) UNIQUE, 
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- enforce both are NULL or both are unique
CREATE UNIQUE INDEX idx_user_auth_phone_unique 
ON user_auth (phone_encrypted) 
WHERE phone_encrypted IS NOT NULL;

CREATE UNIQUE INDEX idx_user_auth_phone_hmac_unique 
ON user_auth (phone_hmac) 
WHERE phone_hmac IS NOT NULL;

