package user

import (
	"context"
	"project-k/internals/security"
	"project-k/pkg/apperror"

	"github.com/google/uuid"
)

type Service struct {
	repo *repository
	tokenSvc *security.TokenService
}

func NewService(repo *repository, tokenSvc *security.TokenService) *Service {
	return &Service{
		repo: repo,
		tokenSvc: tokenSvc,
	}
}

func (s *Service) Register(ctx context.Context, data CreateUserCmd) (uuid.UUID, error) {

	// hash the password
	hashedPassword, err := security.HashPassword(data.PasswordHash)
	if err != nil {
		return uuid.UUID{}, err
	}

	data.PasswordHash = hashedPassword

	id, err := s.repo.CreateUser(ctx, data)
	if err != nil {
		return uuid.UUID{}, nil
	}
	return id, nil
}

func (s *Service) LogIn(ctx context.Context, data LogInUserCmd) (LogInUserResult, error) {
	const op string = "service.user.login"

	u, err := s.repo.GetUserByEmail(ctx, data.Email)
	if err != nil {
		if apperror.IsKind(err, apperror.NotFound) {   // if user not found err, dont return not found, hacker know it
			return LogInUserResult{}, &apperror.Error{
				Kind:    apperror.Unauthorised,
				Op:      op,
				Message: "incorrect email or password",
			}
		}
		return LogInUserResult{}, err // if some other err
	}

	ok, err := security.ComparePassword(data.Password, u.PasswordHash)
	if err != nil || !ok {
		return LogInUserResult{}, &apperror.Error{
			Kind:    apperror.Unauthorised,
			Op:      op,
			Message: "incorrect email or password",
		}
	}
	payload := security.RequestClaims{
		UserID: u.ID.String(),
		Email: u.Email,
	}

	// generate JWT Token with 30 min expiry (as it is just basics)
	token, err :=  s.tokenSvc.GenerateAccessToken(payload)// sceret , expiry is there in token service, we just pass payload
	if err != nil {
		return LogInUserResult{}, &apperror.Error{
			Kind: apperror.Internal,
			Op: op,
			Message: "internal server error",
			Err: err,
		}
	}

	res := LogInUserResult{
		UserID:      u.ID,
		AccessToken: token,
	}
	return res, nil
}

func (s *Service) GetProfile(ctx context.Context, userId uuid.UUID) (User, error) {
	
	u, err := s.repo.GetUserByID(ctx, userId)
	if err != nil {
		return User{}, err
	}
	return u, nil
}

func (s *Service) GetUserByID(ctx context.Context, userID uuid.UUID) (User, error) {

	// const op string = "service.user.get_user_by_id"

	dbUser, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return User{}, err
	}
	return dbUser, nil
}

func (s *Service) IncrementMonitorCount(ctx context.Context, userID uuid.UUID) error {
	return s.repo.IncrementMonitorCount(ctx, userID)
}
