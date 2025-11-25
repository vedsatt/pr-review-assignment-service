package repository

import (
	"context"

	"github.com/Masterminds/squirrel"
	"github.com/vedsatt/pr-review-assignment-service/internal/models"
)

func (r *Repository) SelectUserStats(ctx context.Context) (*models.UserStatsResponse, error) {
	query, args, err := r.builder.
		Select(
			"COUNT(*) as total",
			"COUNT(*) FILTER (WHERE is_active = true) AS active",
			"COUNT(*) FILTER (WHERE is_active = false) AS inactive",
		).
		From("users").
		ToSql()

	if err != nil {
		return nil, wrapDBError(err, "SelectUserStats: build query")
	}

	var total, active, inactive int
	err = r.pool.QueryRow(ctx, query, args...).Scan(&total, &active, &inactive)
	if err != nil {
		return nil, wrapDBError(err, "SelectUserStats: execute query")
	}

	teamQuery, teamArgs, err := r.builder.
		Select("team_name", "COUNT(*) AS users_count").
		From("users").
		GroupBy("team_name").
		ToSql()

	if err != nil {
		return nil, wrapDBError(err, "SelectUserStats: build team query")
	}

	rows, err := r.pool.Query(ctx, teamQuery, teamArgs...)
	if err != nil {
		return nil, wrapDBError(err, "SelectUserStats: execute team query")
	}
	defer rows.Close()

	var teams []models.TeamUsersCount
	for rows.Next() {
		var team models.TeamUsersCount
		err = rows.Scan(&team.TeamName, &team.Users)

		if err != nil {
			return nil, wrapDBError(err, "SelectUserStats: scan team row")
		}

		teams = append(teams, team)
	}

	stats := &models.UserStatsResponse{
		TotalUsers:    total,
		ActiveUsers:   active,
		InactiveUsers: inactive,
		UsersByTeam:   teams,
	}

	return stats, nil
}

func (r *Repository) SelectPullRequestStats(ctx context.Context) (*models.PullRequestsStatsResponse, error) {
	query, args, err := r.builder.
		Select(
			"COUNT(*) as total",
			"COUNT(*) FILTER (WHERE pr_status = 'OPEN') AS open",
			"COUNT(*) FILTER (WHERE pr_status = 'MERGED') AS merged",
		).
		From("pull_requests").
		ToSql()

	if err != nil {
		return nil, wrapDBError(err, "SelectPullRequestStats: scan query")
	}

	var total, open, merged int
	err = r.pool.QueryRow(ctx, query, args...).Scan(&total, &open, &merged)
	if err != nil {
		return nil, wrapDBError(err, "SelectPullRequestStats: scan row")
	}

	stats := &models.PullRequestsStatsResponse{
		TotalPRs:  total,
		OpenPRs:   open,
		MergedPRs: merged,
	}

	return stats, nil
}

func (r *Repository) SelectReviewerStats(ctx context.Context) (*models.ReviewersStatsResponse, error) {
	const defaultLimit = 10
	query, args, err := r.builder.
		Select(
			"u.id",
			"u.user_name",
			"COUNT(prr.reviewer_id) as review_count",
		).
		From("users u").
		LeftJoin("pr_reviewers prr ON u.id = prr.reviewer_id").
		Where(squirrel.Eq{"u.is_active": true}).
		GroupBy("u.id", "u.user_name").
		OrderBy("review_count DESC").
		Limit(defaultLimit).
		ToSql()

	if err != nil {
		return nil, wrapDBError(err, "SelectReviewerStats: build query")
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, wrapDBError(err, "SelectReviewerStats: execute query")
	}
	defer rows.Close()

	var topReviewers []models.Reviewers
	for rows.Next() {
		var reviewer models.Reviewers
		err = rows.Scan(&reviewer.ID, &reviewer.Username, &reviewer.ReviewCount)
		if err != nil {
			return nil, wrapDBError(err, "SelectReviewerStats: scan query row")
		}

		if reviewer.ReviewCount > 0 {
			topReviewers = append(topReviewers, reviewer)
		}
	}

	noReviewQuery, noReviewArgs, err := r.builder.
		Select("u.id").
		From("users u").
		LeftJoin("pr_reviewers prr ON u.id = prr.reviewer_id").
		Where(squirrel.Eq{
			"u.is_active":     true,
			"prr.reviewer_id": nil,
		}).
		ToSql()

	if err != nil {
		return nil, wrapDBError(err, "SelectReviewerStats: build 'no review' query")
	}

	noReviewRows, err := r.pool.Query(ctx, noReviewQuery, noReviewArgs...)
	if err != nil {
		return nil, wrapDBError(err, "SelectReviewerStats: execute 'no review' query")
	}
	defer noReviewRows.Close()

	var noReviewsList []string
	for noReviewRows.Next() {
		var noReviewUser string
		err = noReviewRows.Scan(&noReviewUser)
		if err != nil {
			return nil, wrapDBError(err, "SelectReviewerStats: scan 'no review' row")
		}

		noReviewsList = append(noReviewsList, noReviewUser)
	}

	stats := &models.ReviewersStatsResponse{
		TopReviewers:       topReviewers,
		UsersWithoutReview: noReviewsList,
	}

	return stats, nil
}
