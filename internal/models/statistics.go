package models

type UserStatsResponse struct {
	TotalUsers    int              `json:"total_users"`
	ActiveUsers   int              `json:"active_users"`
	InactiveUsers int              `json:"inactive_users"`
	UsersByTeam   []TeamUsersCount `json:"users_by_team"`
}

type TeamUsersCount struct {
	TeamName string `json:"team_name"`
	Users    int    `json:"users_count"`
}

type PullRequestsStatsResponse struct {
	TotalPRs  int `json:"total_prs"`
	OpenPRs   int `json:"open_prs"`
	MergedPRs int `json:"merged_prs"`
}

type ReviewersStatsResponse struct {
	TopReviewers       []Reviewers `json:"top_reviewers"`
	UsersWithoutReview []string    `json:"users_without_reviews"`
}

type Reviewers struct {
	ID          string `json:"user_id"`
	Username    string `json:"username"`
	ReviewCount int    `json:"review_count"`
}
