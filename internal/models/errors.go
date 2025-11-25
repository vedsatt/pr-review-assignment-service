package models

type ErrDetails struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error ErrDetails `json:"error"`
}

const (
	TeamExistsErr  string = "TEAM_EXISTS"
	UserExistsErr  string = "USER_EXISTS"
	PRExistsErr    string = "PR_EXISTS"
	PRMergedErr    string = "PR_MERGED"
	NotAssignedErr string = "NOT_ASSIGNED"
	NoCandidateErr string = "NO_CANDIDATE"
	NotFoundErr    string = "NOT_FOUND"
	InvalidJSONErr string = "INVALID_JSON"
	InternalErr    string = "NTERNAL_ERROR"
)
