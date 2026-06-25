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

CREATE INDEX idx_contracts_employer_id ON contracts(employer_id);
CREATE INDEX idx_contracts_employee_id ON contracts(employee_id);
CREATE INDEX idx_contracts_application_id ON contracts(application_id);
