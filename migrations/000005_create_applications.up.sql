CREATE TYPE application_status AS ENUM ('PENDING', 'HIRED', 'REJECTED', 'WITHDRAWN', 'COMPLETED');

CREATE TABLE applications (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    gig_id       UUID NOT NULL REFERENCES gigs(id) ON DELETE CASCADE,
    applicant_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status       application_status NOT NULL DEFAULT 'PENDING',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (gig_id, applicant_id)
);

CREATE INDEX idx_applications_gig_id ON applications(gig_id);
CREATE INDEX idx_applications_applicant_id ON applications(applicant_id);