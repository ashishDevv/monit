package user

import (
	"github.com/google/uuid"
)

type User struct {
	ID            uuid.UUID
	Name          string
	Email         string
	PasswordHash  string
	MonitorsCount int32
	IsPaidUser    bool
}

type CreateUserCmd struct {
	Name         string
	Email        string
	PasswordHash string
}

type LogInUserCmd struct {
	Email        string
	PasswordHash string
}