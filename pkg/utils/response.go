package utils

import (
	"encoding/json"
	"errors"
	"net/http"
	"project-k/pkg/apperror"

	"github.com/rs/zerolog/log"
)

type SuccessResponse[T any] struct {
	Success   bool   `json:"success"`
	RequestID string `json:"request_id"`
	Message   string `json:"message"`
	Data      T      `json:"data,omitempty"` // Omit if nil
}

type Error struct {
	Kind    apperror.Kind `json:"kind"` // this is ErrorCode not httpCode, its like VALIDATION_FAILED
	Message string        `json:"message,omitempty"`
	// Details any           `json:"details,omitempty"` // slice or map
}

type ErrorResponse struct {
	Success   bool   `json:"success"`
	RequestID string `json:"request_id"`
	Error     Error  `json:"error"`
}

func WriteJSON[T any](w http.ResponseWriter, status int, reqID string, message string, data T) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	res := SuccessResponse[T]{
		Success: true,
		RequestID: reqID,
		Message: message,
		Data:    data,
	}
	
	if err := json.NewEncoder(w).Encode(res); err != nil {
		log.Error().Err(err).Msg("error in encoding Success Response and sending it to client")
	}
}

func FromAppError(w http.ResponseWriter, reqID string, err error) {

	var appErr *apperror.Error
	if !errors.As(err, &appErr) {
		appErr = &apperror.Error{
			Kind:    apperror.Internal,
			Message: "internal server error",
		}
	}

	httpStatus := apperror.GetHTTPStatus(appErr.Kind)
	WriteError(w, httpStatus, reqID, appErr.Kind, appErr.Message)
}

func WriteError(w http.ResponseWriter, httpStatusCode int, reqID string, code apperror.Kind, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatusCode)

	res := ErrorResponse{
		Success:   false,
		RequestID: reqID,
		Error: Error{
			Kind:    code,
			Message: message,
		},
	}

	if err := json.NewEncoder(w).Encode(res); err != nil {
		log.Error().Err(err).Msg("error in encoding Error Response and sending it to client")
	}
}
