CREATE INDEX IF NOT EXISTS idx_users_team_active ON users(team_name, is_active);

CREATE INDEX IF NOT EXISTS idx_pr_author_id ON pull_requests(author_id);

CREATE INDEX IF NOT EXISTS idx_pr_reviewers_pr_id ON pr_reviewers(pr_id);

CREATE INDEX IF NOT EXISTS idx_pr_reviewers_reviewer_id ON pr_reviewers(reviewer_id);