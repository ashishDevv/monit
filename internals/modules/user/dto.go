package user

type GetProfileResponse struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Email         string `json:"email"`
	MonitorsCount int32  `json:"monitors_count"`
	IsPaidUser    bool   `json:"is_paid_user"`
}

type RegisterRequest struct {
	Name     string `json:"name" validate:"required,gte=3,lte=60"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,gte=8,lte=60"`
}

type RegisterResponse struct {
	UserID string `json:"user_id"`
}

type LogInRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,gte=8,lte=60"`
}

type LogInResponse struct {
	UserID      string `json:"user_id"`
	AccessToken string `json:"access_token"`
}
