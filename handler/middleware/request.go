package middleware

import (
	"encoding/json"
	"io"
	"net/http"

	"gopkg.in/thedevsaddam/govalidator.v1"
)

// ValidatorRules represents validator rules interface for incoming http request model.
type ValidatorRules interface {
	Rules() map[string][]string
}

// ValidatorMessages represents validator messages interface for incoming http request model.
type ValidatorMessages interface {
	Messages() map[string][]string
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

	validatorRules, ok := (out).(ValidatorRules)
	if !ok {
		return nil
	}
	rules := validatorRules.Rules()
	if len(rules) == 0 {
		rules["_"] = []string{}
	}

	opts := govalidator.Options{
		Rules:           rules,
		RequiredDefault: true,
		Data:            out,
	}

	if validatorMessages, ok := (out).(ValidatorMessages); ok {
		opts.Messages = validatorMessages.Messages()
	}

	validationResult := govalidator.New(opts).ValidateStruct()

	if len(validationResult) == 0 {
		return nil
	}

	if parseErr := validationResult.Get("_error"); parseErr != "" {
		return ErrInvalidRequest.New(parseErr)
	}

	paramErrs := make([]RequestParamErrDetails, 0, len(validationResult))
	for param, errs := range validationResult {
		paramErrs = append(paramErrs, RequestParamErrDetails{
			Param:  param,
			Errors: errs,
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
