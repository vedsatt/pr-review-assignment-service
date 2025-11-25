CREATE SEQUENCE pr_id_seq;

CREATE TABLE IF NOT EXISTS pull_requests (
    id VARCHAR(100) PRIMARY KEY,
    pr_name VARCHAR(255) NOT NULL,
    author_id VARCHAR(100) NOT NULL REFERENCES users(id),
    pr_status VARCHAR(6) NOT NULL DEFAULT 'OPEN' CHECK (pr_status IN ('OPEN', 'MERGED')),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    merged_at TIMESTAMPTZ
);