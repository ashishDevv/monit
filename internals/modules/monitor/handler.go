package monitor

import (
	"encoding/json"
	"net/http"
	middle "project-k/internals/middleware"
	"project-k/pkg/apperror"
	"project-k/pkg/utils"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type Handler struct {
	service   *Service
	validator *validator.Validate
	logger    *zerolog.Logger
}

func NewHandler(service *Service, validator *validator.Validate, logger *zerolog.Logger) *Handler {
	return &Handler{
		service:   service,
		validator: validator,
		logger:    logger,
	}
}

func (h *Handler) CreateMonitor(w http.ResponseWriter, r *http.Request) {
	const op string = "handler.monitor.create_monitor"
	ctx := r.Context()
	reqID := middleware.GetReqID(ctx)
	
	reqClaims, ok := middle.UserFromContext(ctx)
	if !ok {
		utils.WriteError(w, http.StatusUnauthorized, reqID, apperror.Unauthorised, "user Unauthorised")
		return
	}
	userID, err := uuid.Parse(reqClaims.UserID)
	if err != nil {
		utils.WriteError(w, http.StatusUnauthorized, reqID, apperror.Unauthorised, "user Unauthorised")
		return
	}

	// decode request body
	var req CreateMonitorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, reqID, apperror.InvalidInput, "invalid request")
		return
	}

	// valideate request body
	if err := h.validator.Struct(req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, reqID, apperror.InvalidInput, "invalid request")
		return
	}

	mID, err := h.service.CreateMonitor(ctx, CreateMonitorCmd{
		UserID:             userID,
		Url:                req.Url,
		IntervalSec:        req.IntervalSec,
		TimeoutSec:         req.TimeoutSec,
		LatencyThresholdMs: req.LatencyThresholdMs,
		ExpectedStatus:     req.ExpectedStatus,
		AlertEmail:         req.AlertEmail,
	})
	if err != nil {
		h.logger.Error().
			Str("op", op).
			Str("req_id", reqID).
			Err(err).
			Msg("create monitor error")
		utils.FromAppError(w, reqID, err)
		return
	}
	utils.WriteJSON(w, http.StatusCreated, reqID, "monitor created sucessfully", CreateMonitorResponse{MonitorID: mID.String()})
}

func (h *Handler) GetMonitor(w http.ResponseWriter, r *http.Request) {
	const op string = "handler.monitor.get_monitor"
	ctx := r.Context()
	reqID := middleware.GetReqID(ctx)
	
	reqClaims, ok := middle.UserFromContext(ctx)
	if !ok {
		utils.WriteError(w, http.StatusUnauthorized, reqID, apperror.Unauthorised, "user Unauthorised")
		return
	}
	userID, err := uuid.Parse(reqClaims.UserID)
	if err != nil {
		utils.WriteError(w, http.StatusUnauthorized, reqID, apperror.Unauthorised, "user Unauthorised")
		return
	}

	mIDStr := chi.URLParam(r, "monitorID")
	monitorID, err := uuid.Parse(mIDStr)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, reqID, apperror.InvalidInput, "invalid input")
		return
	}

	mon, err := h.service.GetMonitor(ctx, userID, monitorID)
	if err != nil {
		h.logger.Error().
			Str("op", op).
			Str("req_id", reqID).
			Err(err).
			Msg("retriving monitor error")
		utils.FromAppError(w, reqID, err)
		return
	}
	m := GetMonitorResponse{
		ID:                 mon.ID.String(),
		Url:                mon.Url,
		AlertEmail:         mon.AlertEmail,
		IntervalSec:        mon.IntervalSec,
		TimeoutSec:         mon.TimeoutSec,
		LatencyThresholdMs: mon.LatencyThresholdMs,
		ExpectedStatus:     mon.ExpectedStatus,
		Enabled:            mon.Enabled,
	}

	utils.WriteJSON(w, http.StatusOK, reqID, "moniter retrived", m)
}

// /monitors?offset=3&limit=10
func (h *Handler) GetAllMonitors(w http.ResponseWriter, r *http.Request) {
	const op string = "handler.monitor.get_all_monitor"
	ctx := r.Context()
	reqID := middleware.GetReqID(ctx)
	
	reqClaims, ok := middle.UserFromContext(ctx)
	if !ok {
		utils.WriteError(w, http.StatusUnauthorized, reqID, apperror.Unauthorised, "user Unauthorised")
		return
	}
	userID, err := uuid.Parse(reqClaims.UserID)
	if err != nil {
		utils.WriteError(w, http.StatusUnauthorized, reqID, apperror.Unauthorised, "user Unauthorised")
		return
	}

	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit, err := strconv.ParseInt(limitStr, 10, 32)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, reqID, apperror.InvalidInput, "invalid input")
		return
	}
	offset, err := strconv.ParseInt(offsetStr, 10, 32)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, reqID, apperror.InvalidInput, "invalid input")
		return
	}
	
	if limit <= 0 {
		limit = 10
	}
	if offset <= 0 {
		offset = 0
	}

	monitors, err := h.service.GetAllMonitors(ctx, userID, int32(limit), int32(offset))
	if err != nil {
		h.logger.Error().
			Str("op", op).
			Str("req_id", reqID).
			Err(err).
			Msg("retriving all monitors error")
		utils.FromAppError(w, reqID, err)
		return
	}
	m := make([]GetMonitorResponse, 0, len(monitors))
	for i := range monitors {
		mon := &monitors[i]
		m = append(m, GetMonitorResponse{
			ID:                 mon.ID.String(),
			Url:                mon.Url,
			IntervalSec:        mon.IntervalSec,
			TimeoutSec:         mon.TimeoutSec,
			LatencyThresholdMs: mon.LatencyThresholdMs,
			ExpectedStatus:     mon.ExpectedStatus,
			Enabled:            mon.Enabled,
			AlertEmail:         mon.AlertEmail,
		})
	}

	resp := GetAllMonitorsResponse{
		UserID:   reqClaims.UserID,
		Limit:    int32(limit),
		Offset:   int32(offset),
		Monitors: m,
	}

	utils.WriteJSON(w, http.StatusOK, reqID, "monitors retrived", resp)
}

// Patch : /monitors/{monitorID}
//
//	{
//		enable: false/true
//	}
func (h *Handler) UpdateMonitorStatus(w http.ResponseWriter, r *http.Request) {
	const op string = "handler.monitor.update_monitor_status"
	ctx := r.Context()
	reqID := middleware.GetReqID(ctx)
	
	reqClaims, ok := middle.UserFromContext(ctx)
	if !ok {
		utils.WriteError(w, http.StatusUnauthorized, reqID, apperror.Unauthorised, "user Unauthorised")
		return
	}
	userID, err := uuid.Parse(reqClaims.UserID)
	if err != nil {
		utils.WriteError(w, http.StatusUnauthorized, reqID, apperror.Unauthorised, "user Unauthorised")
		return
	}
	mIDStr := chi.URLParam(r, "monitorID")
	monitorID, err := uuid.Parse(mIDStr)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, reqID, apperror.InvalidInput, "invalid input")
		return
	}

	// decode request body
	var req UpdateMonitorStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, reqID, apperror.InvalidInput, "invalid request")
		return
	}

	// valideate request body
	if err := h.validator.Struct(req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, reqID, apperror.InvalidInput, "invalid input")
		return
	}

	_, err = h.service.UpdateMonitorStatus(ctx, userID, monitorID, req.Enable)
	if err != nil {
		h.logger.Error().
			Str("op", op).
			Str("req_id", reqID).
			Err(err).
			Msg("updating monitor status error")
		utils.FromAppError(w, reqID, err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, reqID, "Monitor status updated successfully", "ok")
}
