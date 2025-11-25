package service

import (
	"context"

	"github.com/vedsatt/pr-review-assignment-service/internal/models"
)

func (s *Service) GetUsersStatistics(ctx context.Context) (*models.UserStatsResponse, *models.ErrDetails) {
	stats, err := s.repository.SelectUserStats(ctx)
	if err != nil {
		return nil, mapRepositoryError(err)
	}

	return stats, nil
}

func (s *Service) GetPullRequestStatistics(ctx context.Context) (*models.PullRequestsStatsResponse, *models.ErrDetails) {
	stats, err := s.repository.SelectPullRequestStats(ctx)
	if err != nil {
		return nil, mapRepositoryError(err)
	}

	return stats, nil
}

func (s *Service) GetReviewersStatistics(ctx context.Context) (*models.ReviewersStatsResponse, *models.ErrDetails) {
	stats, err := s.repository.SelectReviewerStats(ctx)
	if err != nil {
		return nil, mapRepositoryError(err)
	}

	return stats, nil
}
