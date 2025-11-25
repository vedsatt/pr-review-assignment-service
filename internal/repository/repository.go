package repository

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vedsatt/pr-review-assignment-service/internal/models"
)

type PostgresCfg struct {
	Host     string `env:"POSTGRES_HOST"     env-default:"postgres"`
	Port     string `env:"POSTGRES_PORT"     env-default:"5432"`
	User     string `env:"POSTGRES_USER"     env-default:"postgres"`
	Password string `env:"POSTGRES_PASSWORD" env-default:"postgres"`
	DBName   string `env:"POSTGRES_DB"       env-default:"postgres"`
}

type Repository struct {
	pool    *pgxpool.Pool
	builder squirrel.StatementBuilderType
}

func NewRepository(cfg PostgresCfg) (*Repository, error) {
	addr := net.JoinHostPort(cfg.Host, cfg.Port)
	dataSource := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable",
		cfg.User, cfg.Password, addr, cfg.DBName)

	pool, err := pgxpool.New(context.Background(), dataSource)
	if err != nil {
		return nil, fmt.Errorf("failed to create new pool: %w", err)
	}

	if err = pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	repo := Repository{
		pool:    pool,
		builder: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
	return &repo, nil
}

func wrapDBError(err error, context string) error {
	return fmt.Errorf("database: %s: %w", context, err)
}

func (r *Repository) CloseConnection() {
	r.pool.Close()
}

func (r *Repository) BeginTx(ctx context.Context) (pgx.Tx, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, wrapDBError(err, "BeginTx")
	}

	return tx, nil
}

func (r *Repository) InsertTeam(ctx context.Context, tx pgx.Tx, team models.AddTeamRequest) error {
	query, args, err := r.builder.
		Insert("teams").
		Columns("team_name").
		Values(team.Name).
		ToSql()

	if err != nil {
		return wrapDBError(err, "InsertTeam: build query")
	}

	_, err = tx.Exec(ctx, query, args...)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return errors.New("team unique violation")
		}
		return wrapDBError(err, "InsertTeam: execute query")
	}

	return nil
}

func (r *Repository) InsertTeamMember(ctx context.Context, tx pgx.Tx, member models.TeamMember, teamName string) error {
	query, args, err := r.builder.
		Insert("users").
		Columns("id", "user_name", "is_active", "team_name").
		Values(member.ID, member.Username, member.IsActive, teamName).
		ToSql()

	if err != nil {
		return wrapDBError(err, "InsertTeamMember: build query")
	}

	_, err = tx.Exec(ctx, query, args...)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case "23505":
				return errors.New("user unique violation")
			case "23503":
				return errors.New("not found")
			}
		}
		return wrapDBError(err, "InsertTeamMember: execute query")
	}

	return nil
}

func (r *Repository) SelectTeam(ctx context.Context, teamName string) (*models.Team, error) {
	query, args, err := r.builder.
		Select("id", "user_name", "is_active").
		From("users").
		Where(squirrel.Eq{"team_name": teamName}).
		ToSql()

	if err != nil {
		return nil, wrapDBError(err, "SelectTeam: build query")
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, wrapDBError(err, "SelectTeam: query row")
	}
	defer rows.Close()

	team := &models.Team{
		Name:    teamName,
		Members: make([]models.TeamMember, 0),
	}

	for rows.Next() {
		var member models.TeamMember
		err = rows.Scan(&member.ID, &member.Username, &member.IsActive)
		if err != nil {
			return nil, wrapDBError(err, "SelectTeam: scan")
		}

		team.Members = append(team.Members, member)
	}

	if len(team.Members) == 0 {
		return nil, nil
	}

	return team, nil
}

func (r *Repository) UpdateUserStatus(ctx context.Context, tx pgx.Tx, user models.SetUserStatusRequest) error {
	query, args, err := r.builder.
		Update("users").
		Set("is_active", user.IsActive).
		Where(squirrel.Eq{"id": user.ID}).
		ToSql()

	if err != nil {
		return wrapDBError(err, "UpdateUserStatus: build query")
	}

	result, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return wrapDBError(err, "UpdateUserStatus: execute query")
	}

	if result.RowsAffected() == 0 {
		return errors.New("user not found")
	}

	return nil
}

func (r *Repository) SelectUser(ctx context.Context, userID string) (models.User, error) {
	query, args, err := r.builder.
		Select("id", "user_name", "team_name", "is_active").
		From("users").
		Where(squirrel.Eq{"id": userID}).
		ToSql()

	if err != nil {
		return models.User{}, wrapDBError(err, "SelectUser: build query")
	}

	var user models.User
	err = r.pool.QueryRow(ctx, query, args...).Scan(&user.ID, &user.Username, &user.TeamName, &user.IsActive)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, errors.New("user not found")
		}
		return models.User{}, wrapDBError(err, "SelectUser: query row")
	}

	return user, nil
}

func (r *Repository) SelectUserReviews(ctx context.Context, userID string) ([]*models.PullRequestShort, error) {
	query, args, err := r.builder.
		Select("pr.id", "pr.pr_name", "pr.author_id", "pr.pr_status").
		From("pull_requests pr").
		Join("pr_reviewers prr ON pr.id = prr.pr_id").
		Where(squirrel.Eq{"prr.reviewer_id": userID}).
		ToSql()

	if err != nil {
		return nil, wrapDBError(err, "SelectUserReviews: build query")
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, wrapDBError(err, "SelectUserReviews: execute query")
	}
	defer rows.Close()

	var pullRequests []*models.PullRequestShort
	for rows.Next() {
		var pullRequest models.PullRequestShort

		if err = rows.Scan(&pullRequest.ID, &pullRequest.Name, &pullRequest.AuthorID, &pullRequest.Status); err != nil {
			return nil, wrapDBError(err, "SelectUserReviews: scan row")
		}

		pullRequests = append(pullRequests, &pullRequest)
	}

	return pullRequests, nil
}

func (r *Repository) DeletePullRequestReviewer(ctx context.Context, tx pgx.Tx, prID, reviewerID string) error {
	query, args, err := r.builder.Delete("pr_reviewers").
		Where(squirrel.Eq{
			"pr_id":       prID,
			"reviewer_id": reviewerID,
		}).
		ToSql()

	if err != nil {
		return wrapDBError(err, "DeletePullRequestReviewer: build query")
	}

	if tx != nil {
		_, err = tx.Exec(ctx, query, args...)
	} else {
		_, err = r.pool.Exec(ctx, query, args...)
	}

	if err != nil {
		return wrapDBError(err, "DeletePullRequestReviewer: execute query")
	}

	return nil
}

func (r *Repository) FindAvailableReviewers(ctx context.Context, tx pgx.Tx, user models.User) ([]string, error) {
	const defaultLimit = 10
	query, args, err := r.builder.
		Select("id").
		From("users").
		Where(squirrel.Eq{
			"team_name": user.TeamName,
			"is_active": true,
		}).
		Where(squirrel.NotEq{"id": user.ID}).
		OrderBy("RANDOM()").
		Limit(defaultLimit).
		ToSql()

	if err != nil {
		return nil, wrapDBError(err, "FindAvailableReviewers: build query")
	}

	var rows pgx.Rows
	if tx != nil {
		rows, err = tx.Query(ctx, query, args...)
	} else {
		rows, err = r.pool.Query(ctx, query, args...)
	}

	if err != nil {
		return nil, wrapDBError(err, "FindAvailableReviewers: execute query")
	}
	defer rows.Close()

	var reviewers []string
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			return nil, wrapDBError(err, "FindAvailableReviewers: scan row")
		}
		reviewers = append(reviewers, id)
	}

	return reviewers, nil
}

func (r *Repository) InsertPullRequest(ctx context.Context, tx pgx.Tx, pullRequest models.CreatePRRequest) error {
	query, args, err := r.builder.
		Insert("pull_requests").
		Columns("id", "pr_name", "author_id", "pr_status").
		Values(pullRequest.ID, pullRequest.Name, pullRequest.AuthorID, "OPEN").
		ToSql()

	if err != nil {
		return wrapDBError(err, "InsertPullRequest: build query")
	}

	_, err = tx.Exec(ctx, query, args...)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case "23505":
				return errors.New("pr unique violation")
			case "23503":
				return errors.New("not found")
			}
		}
		return wrapDBError(err, "InsertPullRequest: execute query")
	}

	return nil
}

func (r *Repository) SelectPullRequest(ctx context.Context, pullRequestID string) (*models.PullRequest, error) {
	prQuery, prArgs, err := r.builder.
		Select("id", "pr_name", "author_id", "pr_status").
		From("pull_requests").
		Where(squirrel.Eq{"id": pullRequestID}).
		ToSql()

	if err != nil {
		return nil, wrapDBError(err, "SelectPullRequest: build query")
	}

	var pr models.PullRequest
	err = r.pool.QueryRow(ctx, prQuery, prArgs...).Scan(&pr.ID, &pr.Name, &pr.AuthorID, &pr.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, wrapDBError(err, "SelectPullRequest: query row")
	}

	reviewersQuery, reviewersArgs, err := r.builder.
		Select("reviewer_id").
		From("pr_reviewers").
		Where(squirrel.Eq{"pr_id": pullRequestID}).
		ToSql()

	if err != nil {
		return nil, wrapDBError(err, "SelectPullRequest: build reviewers query")
	}

	rows, err := r.pool.Query(ctx, reviewersQuery, reviewersArgs...)
	if err != nil {
		return nil, wrapDBError(err, "SelectPullRequest: execute reviewers query")
	}
	defer rows.Close()

	var reviewers []string
	for rows.Next() {
		var reviewerID string

		err = rows.Scan(&reviewerID)
		if err != nil {
			return nil, wrapDBError(err, "SelectPullRequest: scan reviewer")
		}

		reviewers = append(reviewers, reviewerID)
	}
	pr.AssignedReviewers = reviewers

	return &pr, nil
}

func (r *Repository) UpdatePullRequestStatus(ctx context.Context, pullRequestID string) error {
	query, args, err := r.builder.
		Update("pull_requests").
		Set("pr_status", "MERGED").
		Set("merged_at", "NOW()").
		Where(squirrel.Eq{"id": pullRequestID}).
		Where(squirrel.Eq{"pr_status": "OPEN"}).
		ToSql()

	if err != nil {
		return wrapDBError(err, "UpdatePullRequestStatus: build query")
	}

	_, err = r.pool.Exec(ctx, query, args...)
	if err != nil {
		return wrapDBError(err, "UpdatePullRequestStatus: execute query")
	}

	return nil
}

func (r *Repository) AssignPullRequestReviewers(ctx context.Context, tx pgx.Tx, pullRequestID string, reviewers []string) error {
	for _, reviewerID := range reviewers {
		query, args, err := r.builder.
			Insert("pr_reviewers").
			Columns("pr_id", "reviewer_id").
			Values(pullRequestID, reviewerID).
			ToSql()

		if err != nil {
			return wrapDBError(err, "AssignPullRequestReviewers: build query")
		}

		_, err = tx.Exec(ctx, query, args...)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23503" {
				return errors.New("not found")
			}
			return wrapDBError(err, "AssignPullRequestReviewers: execute query")
		}
	}

	return nil
}

func (r *Repository) SelectPullRequestReviewers(ctx context.Context, tx pgx.Tx, pullRequestID string) (map[string]bool, error) {
	query, args, err := r.builder.Select("reviewer_id").
		From("pr_reviewers").
		Where(squirrel.Eq{"pr_id": pullRequestID}).
		ToSql()

	if err != nil {
		return nil, wrapDBError(err, "SelectPullRequestReviewers: build query")
	}

	var rows pgx.Rows
	if tx != nil {
		rows, err = tx.Query(ctx, query, args...)
	} else {
		rows, err = r.pool.Query(ctx, query, args...)
	}

	if err != nil {
		return nil, wrapDBError(err, "SelectPullRequestReviewers: execute query")
	}
	defer rows.Close()

	reviewers := make(map[string]bool)
	for rows.Next() {
		var reviewerID string
		err = rows.Scan(&reviewerID)
		if err != nil {
			return nil, wrapDBError(err, "SelectPullRequestReviewers: scan row")
		}

		reviewers[reviewerID] = true
	}

	return reviewers, nil
}

func (r *Repository) ReassignPullRequestReviewer(
	ctx context.Context, tx pgx.Tx, prID, oldReviewerID, authorID, teamName string,
) (string, error) {
	user := models.User{
		ID:       authorID,
		TeamName: teamName,
	}
	reviewers, err := r.FindAvailableReviewers(ctx, tx, user)
	if err != nil {
		return "", err
	}

	currentReviewers, err := r.SelectPullRequestReviewers(ctx, tx, prID)
	if err != nil {
		return "", err
	}

	newReviewerID := ""
	for _, candidateID := range reviewers {
		if candidateID != oldReviewerID && !currentReviewers[candidateID] {
			newReviewerID = candidateID
			break
		}
	}

	if newReviewerID == "" {
		return "", nil
	}

	deleteQuery, deleteArgs, err := r.builder.
		Delete("pr_reviewers").
		Where(squirrel.Eq{
			"pr_id":       prID,
			"reviewer_id": oldReviewerID,
		}).
		ToSql()

	if err != nil {
		return "", wrapDBError(err, "ReassignPullRequestReviewer: build delete query")
	}

	if tx != nil {
		_, err = tx.Exec(ctx, deleteQuery, deleteArgs...)
	} else {
		_, err = r.pool.Exec(ctx, deleteQuery, deleteArgs...)
	}

	if err != nil {
		return "", wrapDBError(err, "ReassignPullRequestReviewer: execute insert query")
	}

	insertQuery, insertArgs, err := r.builder.
		Insert("pr_reviewers").
		Columns("pr_id", "reviewer_id").
		Values(prID, newReviewerID).
		ToSql()

	if err != nil {
		return "", wrapDBError(err, "ReassignPullRequestReviewer: build insert query")
	}

	if tx != nil {
		_, err = tx.Exec(ctx, insertQuery, insertArgs...)
	} else {
		_, err = r.pool.Exec(ctx, insertQuery, insertArgs...)
	}

	if err != nil {
		return "", wrapDBError(err, "ReassignPullRequestReviewer: execute insert query")
	}

	return newReviewerID, nil
}
