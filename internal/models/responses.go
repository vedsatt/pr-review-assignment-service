package models

type AddTeamResponse struct {
	Team Team `json:"team"`
}

type SetUserStatusResponse struct {
	User User `json:"user"`
}

type GetUserReviewsResponse struct {
	UserID       string              `json:"user_id"`
	PullRequests []*PullRequestShort `json:"pull_requests"`
}

type CreatePRResponse struct {
	PullRequest PullRequest `json:"pr"`
}

type MergePullRequestResponse struct {
	PullRequest PullRequest `json:"pr"`
}

type ReassignPullRequestReviewerResponse struct {
	PullRequest PullRequest `json:"pr"`
	ReplacedBy  string      `json:"replaced_by"`
}
