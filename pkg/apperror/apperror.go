package apperror

import (
	"errors"
	"fmt"
	"runtime/debug"
)


type Error struct {
	Kind Kind       // To make errors understandable 
	Op string       // <layer>.<domain>.<action>
	Err error       // wraped error
	Message string  // client safe and frindly message
	Stack []byte    // stack traces
}

// Error implements the built-in error interface
func (e *Error) Error() string {
	switch {
	case e.Op != "" && e.Err != nil:
		return fmt.Sprintf("%s: %v", e.Op, e.Err)
	case e.Err != nil:
		return e.Err.Error()
	case e.Op != "":
		return e.Op
	default:
		return "unknown error"
	}
}

func (e *Error) Unwrap() error {
	return e.Err
}

func (e *Error) WithMessage(msg string) *Error {
	cp := *e
	cp.Message = msg
	return &cp
}

func (e *Error) WithOp(op string) *Error {
	cp := *e
	cp.Op = op
	return &cp
}

func (e *Error) WithErr(err error) *Error {
	cp := *e
	cp.Err = err

	if cp.Stack == nil && (cp.Kind == Internal || cp.Kind == Dependency) {
		cp.Stack = debug.Stack()
	}

	return &cp
}

func New(kind Kind, op string, err error) *Error {
	e := &Error{
		Kind: kind,
		Op:   op,
		Err:  err,
	}

	if kind == Internal || kind == Dependency {
		e.Stack = debug.Stack()
	}

	return e
}

func IsKind(err error, kind Kind) bool {
	var target *Error
	if errors.As(err, &target) {
		return target.Kind == kind
	}
	return false
}