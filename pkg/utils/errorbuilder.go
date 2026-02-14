package utils

import (
	"context"
	"errors"
	"project-k/pkg/apperror"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rs/zerolog"
)

func WrapRepoError(op string, err error, isNotFoundErrPossible bool, log *zerolog.Logger) error {
	// Context errors
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return &apperror.Error{
			Kind:    apperror.RequestTimeout,
			Op:      op,
			Message: "request cancelled or timed out",
		}
	}

	// if no row present
	if isNotFoundErrPossible && errors.Is(err, pgx.ErrNoRows) {
		return &apperror.Error{
			Kind:    apperror.NotFound,
			Op:      op,
			Message: "resources not found",
		}
	}

	// postgres errors
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		log.Error().
			Str("op", op).
			Str("pg_code", pgErr.Code).
			Str("pg_constraint", pgErr.ConstraintName).
			Str("pg_table", pgErr.TableName).
			Str("pg_detail", pgErr.Detail).
			Err(err).
			Msg("postgres database error")

		return &apperror.Error{
			Kind:    apperror.DatabaseErr,
			Op:      op,
			Message: "internal server error",
			Err:     err,
		}
	}

	// other errors
	return &apperror.Error{
		Kind:    apperror.Internal,
		Op:      op,
		Message: "internal server error",
		Err:     err,
	}
}
