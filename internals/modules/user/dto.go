package user

type GetProfileResponse struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Email         string `json:"email"`
	MonitorsCount int32  `json:"monitors_count"`
	IsPaidUser    bool   `json:"is_paid_user"`
}

type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LogInRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
