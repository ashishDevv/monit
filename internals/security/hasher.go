package security

import (
	"github.com/alexedwards/argon2id"
	"github.com/google/uuid"
)

func NewUUID() uuid.UUID {
	return uuid.New()
}

func HashPassword(password string) (string, error) {
	hash, err :=argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		return "", err
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