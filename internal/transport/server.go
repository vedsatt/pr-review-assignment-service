package transport

import (
	"context"
	"net/http"
	"time"

	"github.com/vedsatt/pr-review-assignment-service/internal/config"
	"github.com/vedsatt/pr-review-assignment-service/internal/models"
	"go.uber.org/zap"
)

type PRService interface {
	AddTeam(ctx context.Context, team models.AddTeamRequest) (*models.Team, *models.ErrDetails)
	GetTeam(ctx context.Context, teamName string) (*models.Team, *models.ErrDetails)
	SetUserStatus(ctx context.Context, userSettings models.SetUserStatusRequest) (models.User, *models.ErrDetails)
	GetUserReviews(ctx context.Context, userID string) ([]*models.PullRequestShort, *models.ErrDetails)
	CreatePullRequest(ctx context.Context, pullRequest models.CreatePRRequest) (*models.PullRequest, *models.ErrDetails)
	MergePullRequest(ctx context.Context, pullRequestID string) (models.PullRequest, *models.ErrDetails)
	ReassignPullRequestReviewer(
		ctx context.Context,
		prSettings models.ReassignPRReviewerRequest,
	) (models.PullRequest, string, *models.ErrDetails)
	GetUsersStatistics(ctx context.Context) (*models.UserStatsResponse, *models.ErrDetails)
	GetPullRequestStatistics(ctx context.Context) (*models.PullRequestsStatsResponse, *models.ErrDetails)
	GetReviewersStatistics(ctx context.Context) (*models.ReviewersStatsResponse, *models.ErrDetails)
}

type server struct {
	httpServer *http.Server
	mux        *http.ServeMux
	service    PRService
}

func StartServer(cfg *config.Config, service PRService) *http.Server {
	mux := http.NewServeMux()

	const defaultTimeout = 5 * time.Second
	httpServer := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           mux,
		ReadHeaderTimeout: defaultTimeout,
	}

	server := &server{
		httpServer: httpServer,
		mux:        mux,
		service:    service,
	}

	server.registerHandlers()

	go func() {
		if err := httpServer.ListenAndServe(); err != nil {
			zap.L().Fatal("failed to star server: %v", zap.Error(err))
		}
	}()

	return httpServer
}

func (s *server) registerHandlers() {
	s.mux.Handle("POST /team/add", logsMiddleware(s.AddTeamHandler))
	s.mux.Handle("GET /team/get", logsMiddleware(s.GetTeamHandler))

	s.mux.Handle("POST /users/setIsActive", logsMiddleware(s.SetUserStatusHandler))
	s.mux.Handle("GET /users/getReview", logsMiddleware(s.GetUserReviewsHandler))

	s.mux.Handle("POST /pullRequest/create", logsMiddleware(s.CreatePullRequestHandler))
	s.mux.Handle("POST /pullRequest/merge", logsMiddleware(s.MergePullRequestHandler))
	s.mux.Handle("POST /pullRequest/reassign", logsMiddleware(s.ReassignPullRequestReviewerHandler))

	s.mux.Handle("GET /statistics/users", logsMiddleware(s.GetUsersStatisticsHandler))
	s.mux.Handle("GET /statistics/pullRequests", logsMiddleware(s.GetPullRequestStatisticsHandler))
	s.mux.Handle("GET /statistics/reviewers", logsMiddleware(s.GetReviewersStatisticHandler))
}

func (s *server) mapServiceErrors(err string) int {
	switch err {
	case models.TeamExistsErr:
		return http.StatusBadRequest
	case models.PRExistsErr, models.PRMergedErr, models.NotAssignedErr, models.NoCandidateErr:
		return http.StatusConflict
	case models.NotFoundErr:
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}
