package user

import (
	"context"
	"project-k/internals/security"

	"github.com/google/uuid"
)

type Service struct {
	repo *repository
}

func NewService(repo *repository) *Service {
	return &Service{
		repo: repo,
	}
}

func (s *Service) Register(ctx context.Context, data CreateUserCmd) (uuid.UUID, error) {

	//check if another user with same email
	// if there , return
	// if not then create a user and return it

	_, err := s.repo.GetUserByEmail(ctx, data.Email)
	if err != nil {
		return uuid.UUID{}, err
	}

	// hash the password
	passwordHash, err := security.HashPassword(data.PasswordHash)
	if err != nil {
		return 
	}

	data.PasswordHash = passwordHash
	id, err := s.repo.CreateUser(ctx, data)
	if err != nil {
		return uuid.UUID{}, nil
	}

	return id, nil
}

func (s *Service) LogIn(ctx context.Context, data LogInUserCmd) (uuid.UUID, error) {

	// first check if user present
		// get user by email, if not got error
	// now compare password , if not match error
	// now create a jwt token 
	// send it to user

	u, err := s.repo.GetUserByEmail(ctx, data.Email)
	if err != nil {
		return uuid.UUID{}, err
	}

	// hash the password
	passwordHash, err := security.HashPassword(data.PasswordHash)
	if err != nil {
		return 
	}

	if u.PasswordHash != passwordHash {
		return // err
	}

	// make a method in security package for generate JWT token

	return // token
}


func (s *Service) GetProfile(ctx context.Context, userId uuid.UUID) (User, error) {
	return s.repo.GetUserByID(ctx, userId)
}


func (r *Service) GetUserByID(ctx context.Context, userID uuid.UUID) (User, error) {

	// const op string = "service.user.get_user_by_id"

	dbUser, err := r.repo.GetUserByID(ctx, userID)
	if err != nil {
		return User{}, err
	}
	return dbUser, nil
}
