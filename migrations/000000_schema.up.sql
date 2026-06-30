-- =============================================================================
-- FULL SCHEMA MIGRATION (UP)
-- Order: 000000 → 000007
-- =============================================================================

-- -----------------------------------------------------------------------------
-- 000000_create_users
-- -----------------------------------------------------------------------------

CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE users (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name               VARCHAR(100) NOT NULL,
    avatar_url         TEXT,
    email              TEXT NOT NULL UNIQUE,
    bio                TEXT,
    is_verified        BOOL NOT NULL DEFAULT FALSE,
    is_available_today BOOL NOT NULL DEFAULT FALSE,
    last_active_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE user_auth (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    google_sub      VARCHAR(255) NOT NULL UNIQUE,
    phone_encrypted TEXT,
    phone_hmac      VARCHAR(255) UNIQUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Enforce both are NULL or both are set and unique
CREATE UNIQUE INDEX idx_user_auth_phone_hmac_unique
    ON user_auth (phone_hmac)
    WHERE phone_hmac IS NOT NULL;

-- -----------------------------------------------------------------------------
-- 000001_create_categories
-- -----------------------------------------------------------------------------

CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE categories (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       VARCHAR(100) NOT NULL,
    slug       VARCHAR(100) NOT NULL UNIQUE,
    embedding  VECTOR(384),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- NOTE: categories has no `status` column; index on slug only
CREATE INDEX idx_categories_slug ON categories(slug);

CREATE TABLE user_categories (
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category_id UUID NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, category_id)
);

-- -----------------------------------------------------------------------------
-- 000002_create_gigs
-- -----------------------------------------------------------------------------

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
    gig_id        UUID PRIMARY KEY REFERENCES gigs(id) ON DELETE CASCADE,
    duration_type duration_type,
    start_date    DATE,
    end_date      DATE,
    pay_amount    NUMERIC(12, 2),
    pay_currency  CHAR(3),
    expires_at    TIMESTAMPTZ
);

CREATE TABLE gig_locations (
    id       UUID PRIMARY KEY,
    gig_id   UUID NOT NULL UNIQUE REFERENCES gigs(id) ON DELETE CASCADE,
    location GEOGRAPHY(Point, 4326) NOT NULL,
    city     TEXT
);

CREATE INDEX idx_gig_locations_geoindex ON gig_locations USING GIST (location);

CREATE TABLE gig_categories (
    gig_id      UUID NOT NULL REFERENCES gigs(id) ON DELETE CASCADE,
    category_id UUID NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    PRIMARY KEY (gig_id, category_id)
);

CREATE INDEX idx_gigs_poster_id  ON gigs(poster_id);
CREATE INDEX idx_gigs_status     ON gigs(status);
CREATE INDEX idx_gigs_created_at ON gigs(created_at DESC);

-- -----------------------------------------------------------------------------
-- 000003_create_user_location
-- -----------------------------------------------------------------------------

CREATE TABLE user_locations (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    location   GEOGRAPHY(Point, 4326) NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_flagged BOOL NOT NULL DEFAULT FALSE
);

CREATE INDEX idx_user_locations_location ON user_locations USING GIST(location);

-- -----------------------------------------------------------------------------
-- 000004_create_user_reputation
-- -----------------------------------------------------------------------------

CREATE TABLE user_reputations (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id            UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    rating_as_employer FLOAT4 NOT NULL DEFAULT 0,
    rating_as_employee FLOAT4 NOT NULL DEFAULT 0,
    rating_count       INT NOT NULL DEFAULT 0
);

-- -----------------------------------------------------------------------------
-- 000005_create_applications
-- -----------------------------------------------------------------------------

CREATE TYPE application_status AS ENUM ('PENDING', 'HIRED', 'REJECTED', 'WITHDRAWN', 'COMPLETED');

CREATE TABLE applications (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    gig_id       UUID NOT NULL REFERENCES gigs(id) ON DELETE CASCADE,
    applicant_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status       application_status NOT NULL DEFAULT 'PENDING',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (gig_id, applicant_id)
);

CREATE INDEX idx_applications_gig_id       ON applications(gig_id);
CREATE INDEX idx_applications_applicant_id ON applications(applicant_id);

-- -----------------------------------------------------------------------------
-- 000006_create_contracts
-- -----------------------------------------------------------------------------

CREATE TYPE contract_status AS ENUM ('ACTIVE', 'AWAITING_APPROVAL', 'COMPLETED', 'DISPUTED', 'CANCELLED');

CREATE TABLE contracts (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    application_id UUID NOT NULL UNIQUE REFERENCES applications(id) ON DELETE CASCADE,
    gig_id         UUID NOT NULL REFERENCES gigs(id) ON DELETE CASCADE,
    employer_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    employee_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status         contract_status NOT NULL DEFAULT 'ACTIVE',
    hired_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at   TIMESTAMPTZ
);

CREATE INDEX idx_contracts_employer_id    ON contracts(employer_id);
CREATE INDEX idx_contracts_employee_id    ON contracts(employee_id);
CREATE INDEX idx_contracts_application_id ON contracts(application_id);

-- -----------------------------------------------------------------------------
-- 000007_create_reviews
-- -----------------------------------------------------------------------------

CREATE TYPE review_role_context AS ENUM ('AS_EMPLOYER', 'AS_EMPLOYEE');

CREATE TABLE reviews (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contract_id  UUID NOT NULL REFERENCES contracts(id) ON DELETE CASCADE,
    reviewer_id  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reviewee_id  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    rating       SMALLINT NOT NULL CHECK (rating >= 1 AND rating <= 5),
    comment      TEXT,
    role_context review_role_context NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (contract_id, reviewer_id)
);

CREATE INDEX idx_reviews_reviewee_id ON reviews(reviewee_id);

-- -----------------------------------------------------------------------------
-- 000008_create_notifications
-- -----------------------------------------------------------------------------

CREATE TABLE notifications (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     uuid NOT NULL REFERENCES users(id),
    type        text NOT NULL,
    ref_gig_id  uuid REFERENCES gigs(id),
    title       text NOT NULL,
    body        text NOT NULL,
    is_read     boolean NOT NULL DEFAULT false,
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_notifications_user_created ON notifications (user_id, created_at DESC);
CREATE INDEX idx_notifications_user_unread ON notifications (user_id) WHERE is_read = false;

-- -----------------------------------------------------------------------------
-- 000008_create_fcm_tokens
-- -----------------------------------------------------------------------------

CREATE TABLE fcm_tokens (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token       text NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (user_id, token)
);

CREATE INDEX idx_fcm_tokens_user ON fcm_tokens (user_id);