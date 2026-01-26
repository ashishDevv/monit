package apperror

type Kind string

var (
	// --- Authentication ---
	InvalidInput   Kind = "invalid_input"
	AlreadyExists  Kind = "already_exist"
	NotFound       Kind = "not_found"
	Conflict       Kind = "conflict"
	Unauthorised   Kind = "unauthorised"
	Forbidden      Kind = "forbidden"
	RequestTimeout Kind = "request_timeout"
	Internal       Kind = "internal"
	Dependency     Kind = "dependency_failure"
	DatabaseErr    Kind = "database_error"
)
