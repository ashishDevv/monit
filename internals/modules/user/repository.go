package user

import (
	"context"
	"errors"
	"project-k/pkg/apperror"
	"project-k/pkg/db"
	"project-k/pkg/utils"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rs/zerolog"
)

type repository struct {
	querier *db.Queries
	logger  *zerolog.Logger
}

func NewRepository(dbExecutor db.DBTX, logger *zerolog.Logger) *repository {
	return &repository{
		querier: db.New(dbExecutor),
		logger:  logger,
	}
}

func (r *repository) CreateUser(ctx context.Context, user CreateUserCmd) (uuid.UUID, error) {
	const op string = "repo.user.create_user"

	id, err := r.querier.CreateUser(ctx, db.CreateUserParams{
		Name:         user.Name,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
	})
	if err == nil {
		return utils.FromPgUUID(id), nil
	}

	// from here we handle errors -> it should be handled as it is as it has a unique constraint

	// Context errors
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return uuid.UUID{}, &apperror.Error{
			Kind:    apperror.RequestTimeout,
			Op:      op,
			Message: "request cancelled or timed out",
		}
	}

	// PostgreSQL errors
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		// Unique constraint → conflict
		if pgErr.Code == "23505" {
			return uuid.UUID{}, &apperror.Error{
				Kind:    apperror.AlreadyExists,
				Op:      op,
				Message: "user already exists",
			}
		}
		r.logger.Error().
			Str("code", pgErr.Code).
			Str("constraint", pgErr.ConstraintName).
			Str("table", pgErr.TableName).
			Err(err).
			Msg("database error")

		// Any other constraint / data issue
		return uuid.UUID{}, &apperror.Error{
			Kind:    apperror.DatabaseErr,
			Op:      op,
			Message: "internal server error",
			Err:     err,
		}
	}

	// Everything else
	return uuid.UUID{}, &apperror.Error{
		Kind:    apperror.Internal,
		Op:      op,
		Message: "internal server error",
		Err:     err,
	}
}

func (r *repository) GetUserByID(ctx context.Context, userID uuid.UUID) (User, error) {
	const op string = "repo.user.get_user_by_id"

	user, err := r.querier.GetUserByID(ctx, utils.ToPgUUID(userID))
	if err == nil {
		return User{
			ID:            utils.FromPgUUID(user.ID),
			Name:          user.Name,
			Email:         user.Email,
			PasswordHash:  user.PasswordHash,
			MonitorsCount: utils.FromPgInt32(user.MonitorsCount),
			IsPaidUser:    utils.FromPgBool(user.IsPaidUser),
		}, nil
	}

	return User{}, utils.WrapRepoError(op, err, true, r.logger)
}

func (r *repository) GetUserByEmail(ctx context.Context, email string) (User, error) {
	const op string = "repo.user.get_user_by_email"

	user, err := r.querier.GetUserByEmail(ctx, email)
	if err == nil {
		return User{
			ID:           utils.FromPgUUID(user.ID),
			Name:         user.Name,
			Email:        user.Email,
			PasswordHash: user.PasswordHash,
		}, nil
	}

	return User{}, utils.WrapRepoError(op, err, true, r.logger)
}

func (r *repository) IncrementMonitorCount(ctx context.Context, userID uuid.UUID) error {
	const op string = "repo.user.increment_monitor_count"

	rows, err := r.querier.IncrementMonitorCount(ctx, utils.ToPgUUID(userID))
	if err == nil {
		if rows == 0 {
			return &apperror.Error{
				Kind:    apperror.Forbidden,
				Op:      op,
				Message: "monitor quota exceed",
			}
		}
		return nil
	}

	return utils.WrapRepoError(op, err, false, r.logger)
}
