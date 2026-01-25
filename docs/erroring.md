You’re right to ask for this **end-to-end**, because error handling only really “clicks” when you see **all layers together**, not fragments.

Below is a **complete, production-style implementation** using:

* pgx + sqlc style repository
* Service layer with business rules
* HTTP controller layer
* A proper `apperror` package
* Correct logging strategy (log once, at the edge)
* Clean mapping from **DB → AppError → HTTP**

I’ll also explain **why** each thing lives where it does.

---

# 🧱 1. Error package (foundation)

This is the **most important piece**.
All layers depend on this.
No layer depends on HTTP directly except the controller.

## `internal/apperror/apperror.go`

```go
package apperror

import "errors"

type Code string

const (
	CodeNotFound   Code = "NOT_FOUND"
	CodeInvalid    Code = "INVALID"
	CodeConflict   Code = "CONFLICT"
	CodeInternal   Code = "INTERNAL"
	CodeUnavailable Code = "UNAVAILABLE"
)

type Error struct {
	Code    Code
	Message string // safe for client
	Err     error  // internal (wrapped)
}

func (e *Error) Error() string {
	return e.Message
}

func (e *Error) Unwrap() error {
	return e.Err
}

// Constructors

func NotFound(msg string) *Error {
	return &Error{
		Code:    CodeNotFound,
		Message: msg,
	}
}

func Invalid(msg string) *Error {
	return &Error{
		Code:    CodeInvalid,
		Message: msg,
	}
}

func Conflict(msg string) *Error {
	return &Error{
		Code:    CodeConflict,
		Message: msg,
	}
}

func Internal(err error) *Error {
	return &Error{
		Code:    CodeInternal,
		Message: "internal server error",
		Err:     err,
	}
}

func Unavailable(err error) *Error {
	return &Error{
		Code:    CodeUnavailable,
		Message: "service temporarily unavailable",
		Err:     err,
	}
}

// Helper
func Is(err error, code Code) bool {
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr.Code == code
	}
	return false
}
```

### Why this design?

* Client only sees `Message`
* Logs get full wrapped error
* Layers don’t need to know HTTP
* Easy to map to HTTP later

---

# 🗄️ 2. Repository layer (DB boundary)

**Responsibility:**

* Talk to DB
* Translate DB/pgx errors → AppErrors
* NO logging

## `internal/repository/user_repository.go`

```go
package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"yourapp/internal/apperror"
	"yourapp/internal/db"
)

type UserRepository struct {
	q *db.Queries
}

func NewUserRepository(q *db.Queries) *UserRepository {
	return &UserRepository{q: q}
}

func (r *UserRepository) GetByID(
	ctx context.Context,
	userID pgtype.UUID,
) (db.User, error) {

	user, err := r.q.GetUserByID(ctx, userID)
	if err == nil {
		return user, nil
	}

	// 1️⃣ No rows (expected case)
	if errors.Is(err, pgx.ErrNoRows) {
		return db.User{}, apperror.NotFound("user not found")
	}

	// 2️⃣ Context errors (timeout / cancel)
	if errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, context.Canceled) {
		return db.User{}, apperror.Unavailable(err)
	}

	// 3️⃣ PostgreSQL-level errors
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return db.User{}, apperror.Internal(
			fmt.Errorf("postgres error %s: %w", pgErr.Code, err),
		)
	}

	// 4️⃣ Scan / unexpected errors
	return db.User{}, apperror.Internal(err)
}
```

### Why NO logging here?

Because:

* Repo doesn’t know request context
* Logging here causes **duplicate logs**
* Repo errors may be handled/recovered later

---

# 🧠 3. Service layer (business logic)

**Responsibility:**

* Apply rules
* Decide meaning of errors
* Still no logging

## `internal/service/user_service.go`

```go
package service

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"

	"yourapp/internal/apperror"
	"yourapp/internal/db"
	"yourapp/internal/repository"
)

type UserService struct {
	repo *repository.UserRepository
}

func NewUserService(repo *repository.UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) GetUser(
	ctx context.Context,
	userID pgtype.UUID,
) (db.User, error) {

	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return db.User{}, err
	}

	// Business rule
	if user.Status == "DELETED" {
		return db.User{}, apperror.NotFound("user not found")
	}

	return user, nil
}
```

### Why still no logging?

Because:

* Service may retry, fallback, or alter behavior
* Logging before final decision creates noise

---

# 🌐 4. HTTP Controller layer (edge of system)

**Responsibility:**

* Convert AppError → HTTP
* Log errors (ONCE)
* Send safe responses

## `internal/http/handler/user_handler.go`

```go
package handler

import (
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5/pgtype"

	"yourapp/internal/apperror"
	"yourapp/internal/service"
)

type UserHandler struct {
	service *service.UserService
	logger  Logger // interface
}

func NewUserHandler(s *service.UserService, l Logger) *UserHandler {
	return &UserHandler{
		service: s,
		logger:  l,
	}
}

func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, err := parseUUIDParam(r, "id")
	if err != nil {
		writeError(w, apperror.Invalid("invalid user id"))
		return
	}

	user, err := h.service.GetUser(ctx, userID)
	if err != nil {
		h.handleError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, user)
}
```

---

## Error handling + logging (centralized)

```go
func (h *UserHandler) handleError(w http.ResponseWriter, r *http.Request, err error) {
	var appErr *apperror.Error
	if !errors.As(err, &appErr) {
		appErr = apperror.Internal(err)
	}

	// LOG ONLY HERE
	if appErr.Code == apperror.CodeInternal ||
		appErr.Code == apperror.CodeUnavailable {
		h.logger.Error(
			"request failed",
			"path", r.URL.Path,
			"error", appErr.Err,
		)
	}

	writeError(w, appErr)
}
```

---

## AppError → HTTP mapping

```go
func writeError(w http.ResponseWriter, err *apperror.Error) {
	status := http.StatusInternalServerError

	switch err.Code {
	case apperror.CodeNotFound:
		status = http.StatusNotFound
	case apperror.CodeInvalid:
		status = http.StatusBadRequest
	case apperror.CodeConflict:
		status = http.StatusConflict
	case apperror.CodeUnavailable:
		status = http.StatusServiceUnavailable
	}

	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"error": err.Message,
	})
}
```

---

# 🔍 5. Why logging ONLY in controller?

This is **crucial**.

### If you log in repo + service + controller:

* One error = 3 logs
* Noisy
* Hard to trace

### Controller is best because:

* Has request ID
* Has HTTP context
* Knows final outcome
* Logs once, correctly

> **Rule:**
> Log errors **at system boundaries**, not inside business logic.

---

# 🧠 Final mental model (memorize this)

```
SQL error
   ↓
Repository → AppError
   ↓
Service (business meaning)
   ↓
Controller
   ↓
HTTP response + logging
```

---

# ✅ What you now have

* Clean separation of concerns
* No leaked DB errors to clients
* One place for logging
* Easy to test each layer
* Scales to large codebases

This is **real production Go architecture**, not tutorial code.

If you want next:

* middleware-based logging
* request ID propagation
* retry logic for transient DB errors
* structured logging (zap)

Just tell me.
