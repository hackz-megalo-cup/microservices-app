CREATE TABLE call_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    url TEXT NOT NULL,
    status_code INT NOT NULL,
    body_length INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
