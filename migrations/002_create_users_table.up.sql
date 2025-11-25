CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(10) PRIMARY KEY,
    user_name VARCHAR(255),
    team_name VARCHAR(255) NOT NULL REFERENCES teams(team_name),
    is_active BOOLEAN DEFAULT true
);

