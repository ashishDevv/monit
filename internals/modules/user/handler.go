package user

import (
	"encoding/json"
	"net/http"
	middle "project-k/internals/middleware"
	"project-k/pkg/apperror"
	"project-k/pkg/utils"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type Handler struct {
	service   *Service
	validator *validator.Validate
}

func NewHandler(service *Service, validator *validator.Validate) *Handler {
	return &Handler{
		service:   service,
		validator: validator,
	}
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	reqID := middleware.GetReqID(ctx)

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, reqID, apperror.Kind(apperror.InvalidInput), "")
		return
	}
	// valideate request body
	if err := h.validator.Struct(req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, reqID, apperror.Kind(apperror.InvalidInput), "")
		return
	}

	id, err := h.service.Register(ctx, CreateUserCmd{
		Name:         req.Name,
		Email:        req.Email,
		PasswordHash: req.Password,
	})
	if err != nil {
		return
	}

	utils.WriteJSON(w, http.StatusCreated, "user registered", id)
}

func (h *Handler) LogIn(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	reqID := middleware.GetReqID(ctx)

	var req LogInRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, reqID, apperror.Kind(apperror.InvalidInput), "")
		return
	}
	// valideate request body
	if err := h.validator.Struct(req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, reqID, apperror.Kind(apperror.InvalidInput), "")
		return
	}

	token, err := h.service.LogIn(ctx, LogInUserCmd{
		Email:        req.Email,
		PasswordHash: req.Password,
	})
	if err != nil {
		return
	}

	utils.WriteJSON(w, http.StatusCreated, "user registered", token)
}

func (h *Handler) GetProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	reqID := middleware.GetReqID(ctx)

	reqClaims, ok := middle.UserFromContext(ctx)
	if !ok {
		utils.WriteError(w, http.StatusUnauthorized, reqID, apperror.Unauthorised, "")
		return
	}
	userID, err := uuid.Parse(reqClaims.UserID)
	if err != nil {
		utils.WriteError(w, http.StatusUnauthorized, reqID, apperror.Unauthorised, "")
		return
	}

	user, err := h.service.GetProfile(ctx, userID)
	if err != nil {
		return
	}
	u := GetProfileResponse{
		ID:            user.ID.String(),
		Name:          user.Name,
		Email:         user.Email,
		MonitorsCount: user.MonitorsCount,
		IsPaidUser:    user.IsPaidUser,
	}

	utils.WriteJSON(w, http.StatusOK, "profile retrived", u)
}

func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	reqID := middleware.GetReqID(ctx)
	reqClaims, ok := middle.UserFromContext(ctx)
	if !ok {
		utils.WriteError(w, http.StatusUnauthorized, reqID, apperror.Unauthorised, "")
		return
	}

	pathUserID := r.PathValue("userID")

	if reqClaims.UserID == "" || pathUserID == "" {
		utils.WriteError(w, http.StatusBadRequest, reqID, apperror.InvalidInput, "")
	}

	claimUserID := reqClaims.UserID // this is source of truth

	pathUserUUID, err := uuid.Parse(pathUserID)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, reqID, apperror.InvalidInput, "")
		return
	}

	claimUserUUID, err := uuid.Parse(claimUserID)
	if err != nil {
		utils.WriteError(w, http.StatusUnauthorized, reqID, apperror.Unauthorised, "")
		return
	}

	if pathUserUUID != claimUserUUID {
		utils.WriteError(w, http.StatusForbidden, reqID, apperror.Forbidden, "")
		return
	}

	user, err := h.service.GetUserByID(ctx, claimUserUUID)
	if err != nil {
		utils.FromAppError(w, reqID, err)
	}

	utils.WriteJSON(w, http.StatusOK, "Retrived User", user)
}
