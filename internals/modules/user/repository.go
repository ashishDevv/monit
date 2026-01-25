package user

import (
	"context"
	"project-k/pkg/db"
	"project-k/pkg/utils"

	"github.com/google/uuid"
)

type repository struct {
	querier *db.Queries
}

func NewRepository(dbExecutor db.DBTX) *repository {
	return &repository{
		querier: db.New(dbExecutor),
	}
}

func (r *repository) CreateUser(ctx context.Context, user CreateUserCmd) (uuid.UUID, error) {
	// const op string = "repo.user.create_user"

	id, err := r.querier.CreateUser(ctx, db.CreateUserParams{
		Name:         user.Name,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
	})
	if err != nil {
		return uuid.UUID{}, err
	}
	return utils.FromPgUUID(id), nil

	// if err == nil {
	// 	return nil
	// }

	// from here we handle errors

	// Context errors
	// if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
	// 	return &apperror.Error{
	// 		Kind:    apperror.RequestTimeout,
	// 		Op:      op,
	// 		Message: "request timed out",
	// 	}
	// }

	// // PostgreSQL errors
	// var pgErr *pgconn.PgError
	// if errors.As(err, &pgErr) {
	// 	// Unique constraint → conflict
	// 	if pgErr.Code == "23505" {
	// 		return &apperror.Error{
	// 			Kind:    apperror.AlreadyExists,
	// 			Op:      op,
	// 			Message: "user already exists",
	// 			Err:     err,
	// 		}
	// 	}

	// 	// Any other constraint / data issue
	// 	return &apperror.Error{
	// 		Kind:    apperror.InvalidInput,
	// 		Op:      op,
	// 		Message: "invalid user data",
	// 		Err:     err,
	// 	}
	// }

	// // Everything else
	// return &apperror.Error{
	// 	Kind:    apperror.Internal,
	// 	Op:      op,
	// 	Message: "internal server error",
	// 	Err:     err,
	// }
}

func (r *repository) GetUserByID(ctx context.Context, userID uuid.UUID) (User, error) {
	// const op string = "repo.user.get_user_by_id"

	user, err := r.querier.GetUserByID(ctx, utils.ToPgUUID(userID))
	if err != nil {
		return User{}, err
	}
	return User{
		ID:            utils.FromPgUUID(user.ID),
		Name:          user.Name,
		Email:         user.Email,
		PasswordHash:  user.PasswordHash,
		MonitorsCount: utils.FromPgInt32(user.MonitorsCount),
		IsPaidUser:    utils.FromPgBool(user.IsPaidUser),
	}, nil

	// if err == nil {
	// 	return User{
	// 		UserID:      dbUser.UserID.Bytes,
	// 		DisplayName: dbUser.DisplayName,
	// 		Email:       dbUser.Email,
	// 		Status:      dbUser.Status,
	// 		CreatedAt:   dbUser.CreatedAt.Time,
	// 		UpdatedAt:   dbUser.UpdatedAt.Time,
	// 	}, nil
	// }

	// if errors.Is(err, pgx.ErrNoRows) {
	// 	return User{}, &apperror.Error{
	// 		Kind:    apperror.NotFound,
	// 		Op:      op,
	// 		Message: "user not found",
	// 	}
	// }
	// if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
	// 	// Timeout or request cancelled
	// 	return User{}, &apperror.Error{
	// 		Kind:    apperror.RequestTimeout,
	// 		Op:      op,
	// 		Message: "request timed out",
	// 	}
	// }

	// var pgErr *pgconn.PgError
	// if errors.As(err, &pgErr) {
	// 	// PostgreSQL-specific error
	// 	// pgErr.Code, pgErr.Message, pgErr.ConstraintName
	// 	// log this
	// 	return User{}, &apperror.Error{
	// 		Kind:    apperror.Internal,
	// 		Op:      op,
	// 		Message: "internal server error",
	// 		Err:     err,
	// 	}
	// }
	// Unknown / scan / internal error
	// log this
	// return User{}, &apperror.Error{
	// 	Kind:    apperror.Internal,
	// 	Op:      op,
	// 	Message: "internal server error",
	// 	Err:     err,
	// }
}

func (r *repository) GetUserByEmail(ctx context.Context, email string) (User, error) {
	// const op string = "repo.user.get_user_by_id"

	user, err := r.querier.GetUserByEmail(ctx, email)
	if err != nil {
		return User{}, err
	}
	return User{
		ID:            utils.FromPgUUID(user.ID),
		Name:          user.Name,
		Email:         user.Email,
		PasswordHash:  user.PasswordHash,
	}, nil
}

func (r *repository) IncrementMonitorCount(ctx context.Context, userID uuid.UUID) error {

	return r.querier.IncrementMonitorCount(ctx, utils.ToPgUUID(userID))
}
