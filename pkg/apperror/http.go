package apperror

import (
	"errors"
	"net/http"
)

func HTTPStatus(err error) int {
	var e *Error
	if !errors.As(err, &e) {
		return http.StatusInternalServerError
	}

	switch e.Kind {
	case InvalidInput:
		return http.StatusBadRequest
	case NotFound:
		return http.StatusNotFound
	case Conflict:
		return http.StatusConflict
	case Unauthorised:
		return http.StatusUnauthorized
	case Forbidden:
		return http.StatusForbidden
	case Dependency:
		return http.StatusBadGateway
	case Internal:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

func GetHTTPStatus(kind Kind) int {

	switch kind {
	case InvalidInput:
		return http.StatusBadRequest
	case NotFound:
		return http.StatusNotFound
	case Conflict:
		return http.StatusConflict
	case Unauthorised:
		return http.StatusUnauthorized
	case Forbidden:
		return http.StatusForbidden
	case Dependency:
		return http.StatusBadGateway
	case Internal:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}
