CREATE TABLE greetings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    external_status INT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
