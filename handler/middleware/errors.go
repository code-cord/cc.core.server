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

	// server errors 1xxx.
	errServerPing = 1000
)

// Custom error (aka unexpected error).
var (
	ErrCustom = Error{
		Code:    errCodeCustom,
		Message: "unexpected error",
	}
)

// Server error.
var (
	ErrServerPing = Error{
		Code:    errServerPing,
		Message: "could not ping server",
	}
)

// Error represents generic model for error.
type Error struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
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
