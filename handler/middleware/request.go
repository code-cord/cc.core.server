package middleware

import (
	"encoding/json"
	"io"
	"net/http"

	validation "github.com/go-ozzo/ozzo-validation/v4"
)

// Validator represents validator interface.
type Validator interface {
	Validate() error
}

// WriteJSONResponse writes JSON encoded body to http response.
func WriteJSONResponse(w http.ResponseWriter, statusCode int, body interface{}) {
	w.WriteHeader(statusCode)
	w.Header().Set("Content-Type", "application/json")

	if body != nil {
		json.NewEncoder(w).Encode(body)
	}
}

// ParseJSONRequest parses JSON encoded http request body.
func ParseJSONRequest(r *http.Request, out interface{}) error {
	err := json.NewDecoder(r.Body).Decode(out)
	if err != nil && err != io.EOF {
		return ErrInvalidRequest.New(err.Error())
	}

	validator, ok := out.(Validator)
	if !ok {
		return nil
	}

	err = validator.Validate()
	if err == nil {
		return nil
	}

	validationErrs, ok := err.(validation.Errors)
	if !ok {
		return ErrInvalidRequest.New(err.Error())
	}

	paramErrs := make([]RequestParamErrDetails, 0, len(validationErrs))
	for param, errs := range validationErrs {
		paramErrs = append(paramErrs, RequestParamErrDetails{
			Param:  param,
			Errors: []string{errs.Error()},
		})
	}

	return ErrInvalidRequestParam.New(paramErrs)
}

// UpgradeRequestToSSE upgrades existing HTTP request to SSE.
func UpgradeRequestToSSE(w http.ResponseWriter, origin string) {
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
}
