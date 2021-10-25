package middleware

import "fmt"

const (
	// custom errors.
	errCodeCustom = 0

	// request errors.
	/*errCodeInvalidRequest      = 1000
	errCodeInvalidRequestParam = 1001
	errCodeSSEUpgrade          = 1002
	errCodeSSESend             = 1003
	errCodeAuth                = 1004

	// stream errors.
	errStreamStart           = 2000
	errStreamStop            = 2001
	errJoinParticipant       = 2002
	errAcceptParticipantJoin = 2003
	errRejectParticipantJoin = 2004*/

	// request errors 1xxx.
	errCodeInvalidRequest      = 1000
	errCodeInvalidRequestParam = 1001

	// server errors 2xxx.
	errCodeServerPing   = 2000
	errCodeCreateStream = 2001
)

// Custom error (aka unexpected error).
var (
	ErrCustom = Error{
		Code:    errCodeCustom,
		Message: "unexpected error",
	}
)

// Request error.
var (
	ErrInvalidRequest = Error{
		Code:    errCodeInvalidRequest,
		Message: "invalid request",
	}
	ErrInvalidRequestParam = Error{
		Code:    errCodeInvalidRequestParam,
		Message: "invalid param",
	}
)

// Server error.
var (
	ErrServerPing = Error{
		Code:    errCodeServerPing,
		Message: "could not ping server",
	}
	ErrCreateStream = Error{
		Code:    errCodeCreateStream,
		Message: "could not create stream",
	}
)

// Error represents generic model for error.
type Error struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// RequestParamErrDetails describes request param error details.
type RequestParamErrDetails struct {
	Param  string   `json:"param"`
	Errors []string `json:"errors"`
}

// New creates a new copy of Error.
func (e Error) New(details interface{}) Error {
	return Error{
		Code:    e.Code,
		Message: e.Message,
		Details: details,
	}
}

// Error returns human-readable error message.
func (e Error) Error() string {
	return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Details)
}
