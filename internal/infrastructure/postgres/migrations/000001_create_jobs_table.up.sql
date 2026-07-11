CREATE TYPE job_status AS ENUM (
    'PENDING',
    'RUNNING',
    'SUCCEEDED',
    'FAILED'
);

CREATE TABLE jobs (
                      id UUID PRIMARY KEY,
                      prompt TEXT NOT NULL,
                      status job_status NOT NULL DEFAULT 'PENDING',
                      result TEXT,
                      failure_reason TEXT,
                      created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                      updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);