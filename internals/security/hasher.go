package security

import (
	"project-k/pkg/apperror"

	"github.com/alexedwards/argon2id"
	"github.com/google/uuid"
)

func NewUUID() uuid.UUID {
	return uuid.New()
}

func HashPassword(password string) (string, error) {
	const op string = "infra.security.hash_password"

	hash, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		return "", &apperror.Error{
			Kind: apperror.Dependency,
			Op: op,
			Message: "internal server error",
			Err: err,
		}
	}
	return hash, nil
}

func ComparePassword(password, hash string) (bool, error) {
	ok, err := argon2id.ComparePasswordAndHash(password, hash)
	if err != nil {
		return false, err
	}
	return ok, nil
}