package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/vedsatt/pr-review-assignment-service/internal/models"
	"go.uber.org/zap"
)

type Repository interface {
	BeginTx(ctx context.Context) (pgx.Tx, error)
	InsertTeam(ctx context.Context, tx pgx.Tx, team models.AddTeamRequest) error
	InsertTeamMember(ctx context.Context, tx pgx.Tx, member models.TeamMember, teamName string) error
	SelectTeam(ctx context.Context, teamName string) (*models.Team, error)
	UpdateUserStatus(ctx context.Context, tx pgx.Tx, user models.SetUserStatusRequest) error
	SelectUser(ctx context.Context, userID string) (models.User, error)
	FindAvailableReviewers(ctx context.Context, tx pgx.Tx, user models.User) ([]string, error)
	SelectUserReviews(ctx context.Context, userID string) ([]*models.PullRequestShort, error)
	DeletePullRequestReviewer(ctx context.Context, tx pgx.Tx, prID, reviewerID string) error
	InsertPullRequest(ctx context.Context, tx pgx.Tx, pullRequest models.CreatePRRequest) error
	SelectPullRequest(ctx context.Context, pullRequestID string) (*models.PullRequest, error)
	UpdatePullRequestStatus(ctx context.Context, pullRequestID string) error
	AssignPullRequestReviewers(ctx context.Context, tx pgx.Tx, pullRequestID string, reviewers []string) error
	ReassignPullRequestReviewer(
		ctx context.Context,
		tx pgx.Tx,
		prID, reviewerID, authorID, teamName string,
	) (string, error)
	SelectUserStats(ctx context.Context) (*models.UserStatsResponse, error)
	SelectPullRequestStats(ctx context.Context) (*models.PullRequestsStatsResponse, error)
	SelectReviewerStats(ctx context.Context) (*models.ReviewersStatsResponse, error)
}

type Service struct {
	repository Repository
}

func NewService(repo Repository) *Service {
	return &Service{
		repository: repo,
	}
}

func mapRepositoryError(err error) *models.ErrDetails {
	internal := &models.ErrDetails{
		Code:    models.InternalErr,
		Message: "service unavailable, try again later",
	}

	if isDatabaseError(err) {
		zap.L().Error("server error", zap.Error(err), zap.String("type", "technical"))
		return internal
	}

	var businessErr *models.ErrDetails
	errMsg := err.Error()
	switch {
	case strings.Contains(errMsg, "team unique violation"):
		businessErr = &models.ErrDetails{Code: models.TeamExistsErr, Message: "team already exists"}
	case strings.Contains(errMsg, "user unique violation"):
		businessErr = &models.ErrDetails{Code: models.UserExistsErr, Message: "user already exists"}
	case strings.Contains(errMsg, "pr unique violation"):
		businessErr = &models.ErrDetails{Code: models.PRExistsErr, Message: "pull request already exists"}
	case strings.Contains(errMsg, "foreign key violation"), strings.Contains(errMsg, "not found"):
		businessErr = &models.ErrDetails{Code: models.NotFoundErr, Message: "resource not found"}
	default:
		zap.L().Warn("unknown business error",
			zap.Error(err),
			zap.String("type", "business_unknown"),
		)
		return internal
	}

	zap.L().Info("business logic error", zap.Error(err), zap.String("type", "business"))

	return businessErr
}

func isDatabaseError(err error) bool {
	return strings.HasPrefix(err.Error(), "database:")
}

func (s *Service) AddTeam(ctx context.Context, team models.AddTeamRequest) (*models.Team, *models.ErrDetails) {
	if team.Name == "" {
		zap.L().Info("business logic error",
			zap.Error(errors.New("AddTeam: empty team_name")),
			zap.String("type", "business"))

		return nil, &models.ErrDetails{Code: models.NotFoundErr, Message: "empty team_name"}
	}

	if len(team.Members) == 0 {
		return nil, &models.ErrDetails{Code: models.NotFoundErr, Message: "empty members"}
	}

	teamExists, err := s.repository.SelectTeam(ctx, team.Name)
	if err != nil {
		return nil, mapRepositoryError(err)
	}

	if teamExists != nil && len(teamExists.Members) > 0 {
		return nil, mapRepositoryError(err)
	}

	tx, err := s.repository.BeginTx(ctx)
	if err != nil {
		return nil, mapRepositoryError(err)
	}
	defer func() {
		if err = tx.Rollback(ctx); err != nil {
			zap.L().Error("transaction rollbock error",
				zap.Error(fmt.Errorf("AddTeam: failed to rollback tx: %w", err)),
				zap.String("type", "technical"))
		}
	}()

	err = s.repository.InsertTeam(ctx, tx, team)
	if err != nil {
		return nil, mapRepositoryError(err)
	}

	for _, member := range team.Members {
		if member.ID == "" {
			zap.L().Info("business logic error",
				zap.Error(errors.New("AddTeam: empty user_id")),
				zap.String("type", "business"))

			return nil, &models.ErrDetails{Code: models.NotFoundErr, Message: "empty user_id"}
		}

		err = s.repository.InsertTeamMember(ctx, tx, member, team.Name)
		if err != nil {
			return nil, mapRepositoryError(err)
		}
	}

	if err = tx.Commit(ctx); err != nil {
		zap.L().Error("server error",
			zap.Error(fmt.Errorf("AddTeam: Commit tx: %w", err)),
			zap.String("type", "technical"))

		return nil, &models.ErrDetails{Code: models.InternalErr, Message: "service unavailable, try again later"}
	}

	teamResp, err := s.repository.SelectTeam(ctx, team.Name)
	if err != nil || teamResp == nil {
		return nil, mapRepositoryError(err)
	}

	return teamResp, nil
}

func (s *Service) GetTeam(ctx context.Context, teamName string) (*models.Team, *models.ErrDetails) {
	if teamName == "" {
		zap.L().Error("business logic error",
			zap.Error(errors.New("empty team_name")),
			zap.String("type", "business"))

		return nil, &models.ErrDetails{Code: models.NotFoundErr, Message: "empty team_name"}
	}

	team, err := s.repository.SelectTeam(ctx, teamName)
	if err != nil {
		return nil, mapRepositoryError(err)
	}

	if team == nil {
		zap.L().Error("business logic error",
			zap.Error(errors.New("team not found")),
			zap.String("type", "business"))
		return nil, &models.ErrDetails{Code: models.NotFoundErr, Message: "team not found"}
	}

	return team, nil
}

func (s *Service) SetUserStatus(ctx context.Context, userSettings models.SetUserStatusRequest) (models.User, *models.ErrDetails) {
	if userSettings.ID == "" {
		zap.L().Error("business logic error",
			zap.Error(errors.New("SetUserStatus: empty user_id")),
			zap.String("type", "business"))

		return models.User{}, &models.ErrDetails{Code: models.NotFoundErr, Message: "empty user_id"}
	}

	tx, err := s.repository.BeginTx(ctx)
	if err != nil {
		return models.User{}, mapRepositoryError(err)
	}
	defer func() {
		if err = tx.Rollback(ctx); err != nil {
			zap.L().Error("transaction rollbock error",
				zap.Error(fmt.Errorf("SetUserStatus: failed to rollback tx: %w", err)),
				zap.String("type", "technical"))
		}
	}()

	err = s.repository.UpdateUserStatus(ctx, tx, userSettings)
	if err != nil {
		return models.User{}, mapRepositoryError(err)
	}

	if !userSettings.IsActive {
		if deactivateErr := s.deactivateUser(ctx, tx, userSettings); deactivateErr != nil {
			return models.User{}, deactivateErr
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return models.User{}, mapRepositoryError(err)
	}

	user, err := s.repository.SelectUser(ctx, userSettings.ID)
	if err != nil {
		return models.User{}, mapRepositoryError(err)
	}

	return user, nil
}

func (s *Service) deactivateUser(ctx context.Context, tx pgx.Tx, userSettings models.SetUserStatusRequest) *models.ErrDetails {
	user, err := s.repository.SelectUser(ctx, userSettings.ID)
	if err != nil {
		return mapRepositoryError(err)
	}

	pullRequests, err := s.repository.SelectUserReviews(ctx, userSettings.ID)
	if err != nil {
		return mapRepositoryError(err)
	}

	for _, pr := range pullRequests {
		if pr.Status != "MERGED" {
			newReviewer, serviceErr := s.tryReassignReviewer(ctx, tx, pr.ID,
				userSettings.ID, pr.AuthorID, user.TeamName)

			if serviceErr != nil {
				return serviceErr
			}

			if newReviewer == "" {
				err = s.repository.DeletePullRequestReviewer(ctx, tx, pr.ID, userSettings.ID)
				if err != nil {
					return mapRepositoryError(err)
				}
			}
		}
	}

	return nil
}

func (s *Service) GetUserReviews(ctx context.Context, userID string) ([]*models.PullRequestShort, *models.ErrDetails) {
	if userID == "" {
		zap.L().Error("business logic error",
			zap.Error(errors.New("GetUserReviews: user not found")),
			zap.String("type", "business"))

		return nil, &models.ErrDetails{Code: models.NotFoundErr, Message: "resource not found"}
	}

	reviews, err := s.repository.SelectUserReviews(ctx, userID)
	if err != nil {
		return nil, mapRepositoryError(err)
	}

	return reviews, nil
}

func (s *Service) CreatePullRequest(
	ctx context.Context,
	pullRequest models.CreatePRRequest,
) (*models.PullRequest, *models.ErrDetails) {
	if pullRequest.AuthorID == "" {
		zap.L().Error("business logic error",
			zap.Error(errors.New("CreatePullRequest: empty author_id")),
			zap.String("type", "business"))

		return nil, &models.ErrDetails{Code: models.NotFoundErr, Message: "empty author_id"}
	}

	if pullRequest.ID == "" {
		zap.L().Error("business logic error",
			zap.Error(errors.New("CreatePullRequest: empty pull_request_id")),
			zap.String("type", "business"))

		return nil, &models.ErrDetails{Code: models.NotFoundErr, Message: "empty pull_request_id"}
	}

	exists, err := s.repository.SelectPullRequest(ctx, pullRequest.ID)
	if err != nil {
		return nil, mapRepositoryError(err)
	}

	if exists != nil {
		zap.L().Error("business logic error",
			zap.Error(errors.New("CreatePullRequest: pull request already exists")),
			zap.String("type", "business"))

		return nil, &models.ErrDetails{Code: models.PRExistsErr, Message: "pull request already exists"}
	}

	user, err := s.repository.SelectUser(ctx, pullRequest.AuthorID)
	if err != nil {
		return nil, mapRepositoryError(err)
	}

	tx, err := s.repository.BeginTx(ctx)
	if err != nil {
		return nil, mapRepositoryError(err)
	}
	defer func() {
		if err = tx.Rollback(ctx); err != nil {
			zap.L().Error("transaction rollbock error",
				zap.Error(fmt.Errorf("CreatePullRequest: failed to rollback tx: %w", err)),
				zap.String("type", "technical"))
		}
	}()

	err = s.repository.InsertPullRequest(ctx, tx, pullRequest)
	if err != nil {
		return nil, mapRepositoryError(err)
	}

	allReviewers, err := s.repository.FindAvailableReviewers(ctx, tx, user)
	if err != nil {
		return nil, mapRepositoryError(err)
	}

	if len(allReviewers) != 0 {
		var limitedReviewers []string
		if len(allReviewers) > 2 {
			limitedReviewers = allReviewers[:2]
		} else {
			limitedReviewers = allReviewers
		}

		err = s.repository.AssignPullRequestReviewers(ctx, tx, pullRequest.ID, limitedReviewers)
		if err != nil {
			return nil, mapRepositoryError(err)
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, mapRepositoryError(err)
	}

	pr, err := s.repository.SelectPullRequest(ctx, pullRequest.ID)
	if err != nil {
		return nil, mapRepositoryError(err)
	}

	return pr, nil
}

func (s *Service) MergePullRequest(ctx context.Context, pullRequestID string) (models.PullRequest, *models.ErrDetails) {
	if pullRequestID == "" {
		zap.L().Error("business logic error",
			zap.Error(errors.New("CreatePullRequest: pull request already exists")),
			zap.String("type", "business"))

		return models.PullRequest{}, &models.ErrDetails{Code: models.NotFoundErr, Message: "resource not found"}
	}

	if err := s.repository.UpdatePullRequestStatus(ctx, pullRequestID); err != nil {
		return models.PullRequest{}, mapRepositoryError(err)
	}

	pr, err := s.repository.SelectPullRequest(ctx, pullRequestID)
	if err != nil {
		return models.PullRequest{}, mapRepositoryError(err)
	}

	return *pr, nil
}

func (s *Service) ReassignPullRequestReviewer(
	ctx context.Context,
	prSettings models.ReassignPRReviewerRequest,
) (models.PullRequest, string, *models.ErrDetails) {
	if prSettings.OldReviewerID == "" || prSettings.PullRequestID == "" {
		zap.L().Error("business logic error",
			zap.Error(errors.New("ReassignPullRequestReviewer: reviewer_id or pull_request_id is empty")),
			zap.String("type", "business"))

		return models.PullRequest{}, "", &models.ErrDetails{Code: models.NotFoundErr, Message: "resource not found"}
	}

	assignedPR, err := s.repository.SelectPullRequest(ctx, prSettings.PullRequestID)
	if err != nil {
		return models.PullRequest{}, "", mapRepositoryError(err)
	}

	if assignedPR == nil {
		zap.L().Error("business logic error",
			zap.Error(errors.New("ReassignPullRequestReviewer: user not assigned on pull request")),
			zap.String("type", "business"))

		return models.PullRequest{}, "",
			&models.ErrDetails{Code: models.NotAssignedErr, Message: "user not assigned on pull request"}
	}

	if assignedPR.Status == "MERGED" {
		zap.L().Error("business logic error",
			zap.Error(errors.New("ReassignPullRequestReviewer: can't reassign reviewer on merged pull request")),
			zap.String("type", "business"))

		return models.PullRequest{}, "",
			&models.ErrDetails{Code: models.PRMergedErr, Message: "can't reassign reviewer on merged pull request"}
	}

	tx, err := s.repository.BeginTx(ctx)
	if err != nil {
		return models.PullRequest{}, "", mapRepositoryError(err)
	}

	defer func() {
		if err = tx.Rollback(ctx); err != nil {
			zap.L().Error("transaction rollbock error",
				zap.Error(fmt.Errorf("ReassignPullRequestReviewer: failed to rollback tx: %w", err)),
				zap.String("type", "technical"))
		}
	}()

	user, err := s.repository.SelectUser(ctx, prSettings.OldReviewerID)
	if err != nil {
		return models.PullRequest{}, "", mapRepositoryError(err)
	}

	replacedBy, err := s.repository.ReassignPullRequestReviewer(
		ctx, tx, prSettings.PullRequestID, prSettings.OldReviewerID, assignedPR.AuthorID, user.TeamName)

	if err != nil {
		return models.PullRequest{}, "", mapRepositoryError(err)
	}

	if replacedBy == "" {
		zap.L().Info("business logic error",
			zap.Error(errors.New("ReassignPullRequestReviewer: no available reviewers")),
			zap.String("type", "business"))

		return models.PullRequest{}, "", &models.ErrDetails{Code: models.NoCandidateErr, Message: "no available reviewers"}
	}

	if err = tx.Commit(ctx); err != nil {
		return models.PullRequest{}, "", mapRepositoryError(err)
	}

	pr, err := s.repository.SelectPullRequest(ctx, prSettings.PullRequestID)
	if err != nil {
		return models.PullRequest{}, "", mapRepositoryError(err)
	}

	return *pr, replacedBy, nil
}

func (s *Service) tryReassignReviewer(
	ctx context.Context, tx pgx.Tx, prID, oldReviewerID, authorID, teamName string,
) (string, *models.ErrDetails) {
	replacedBy, err := s.repository.ReassignPullRequestReviewer(
		ctx, tx, prID, oldReviewerID, authorID, teamName)

	if err != nil {
		return "", mapRepositoryError(err)
	}

	return replacedBy, nil
}
