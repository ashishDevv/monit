package utils

import (
	"encoding/json"
	"errors"
	"net/http"
	"project-k/pkg/apperror"
)

type SuccessResponse[T any] struct {
	Success   bool   `json:"success"`
	RequestID string `json:"request_id"`
	Message   string `json:"message"`
	Data      T      `json:"data,omitempty"` // Omit if nil
}

type ErrorResponse struct {
	Success   bool   `json:"success"`
	RequestID string `json:"request_id"`
	Error     struct {
		Kind    apperror.Kind `json:"kind"` // this is ErrorCode not httpCode, its like VALIDATION_FAILED
		Message string        `json:"message,omitempty"`
		// Details any    `json:"details,omitempty"` // slice or map
	} `json:"error"`
}

func WriteJSON[T any](w http.ResponseWriter, status int, message string, data T) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	res := SuccessResponse[T]{
		Success: true,
		Message: message,
		Data:    data,
	}
	_ = json.NewEncoder(w).Encode(res)
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
		Success: false,
	}
	res.RequestID = reqID
	res.Error.Kind = code
	res.Error.Message = message

	_ = json.NewEncoder(w).Encode(res)
}
