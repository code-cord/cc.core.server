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
	errCodeSSEUpgrade          = 1002
	errCodeAuth                = 1003

	// server errors 2xxx.
	errCodeServerPing   = 2000
	errCodeCreateStream = 2001
	errCodeFinishStream = 2002
	errCodeUpdateStream = 2003

	// stream errors 3xxx.
	errCodeJoinStream              = 3000
	errCodeFetchStreamParticipants = 3001
	errCodeDecideParticipantJoin   = 3002
	errCodeGenerateToken           = 3003
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
	ErrSSEUpgrade = Error{
		Code:    errCodeSSEUpgrade,
		Message: "could not upgrade SSE connection",
	}
	ErrAuth = Error{
		Code:    errCodeAuth,
		Message: "could not authorize request",
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
	ErrFinishStream = Error{
		Code:    errCodeFinishStream,
		Message: "could not finish stream",
	}
	ErrUpdateStream = Error{
		Code:    errCodeUpdateStream,
		Message: "could not update stream info",
	}
)

// Stream error.
var (
	ErrJoinStream = Error{
		Code:    errCodeJoinStream,
		Message: "could not join the stream",
	}
	ErrFetchStreamParticipants = Error{
		Code:    errCodeFetchStreamParticipants,
		Message: "could not fetch list of stream participants",
	}
	ErrDecideParticipantJoin = Error{
		Code:    errCodeDecideParticipantJoin,
		Message: "could not change participant join status",
	}
	ErrGenerateAccessToken = Error{
		Code:    errCodeGenerateToken,
		Message: "could not generate access token",
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
