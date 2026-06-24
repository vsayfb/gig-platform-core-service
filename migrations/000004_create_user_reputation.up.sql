CREATE TABLE user_reputations (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id            UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    rating_as_employer FLOAT4 NOT NULL DEFAULT 0,
    rating_as_employee FLOAT4 NOT NULL DEFAULT 0,
    rating_count       INT NOT NULL DEFAULT 0
);
