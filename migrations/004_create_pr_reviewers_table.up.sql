CREATE TABLE IF NOT EXISTS pr_reviewers (
    pr_id VARCHAR(100) NOT NULL REFERENCES pull_requests(id),
    reviewer_id VARCHAR(10) REFERENCES users(id),
    PRIMARY KEY (pr_id, reviewer_id)
);