package middleware

import (
	"encoding/json"
	"net/http"
)

// WriteJSONResponse writes JSON encoded body to http response.
func WriteJSONResponse(w http.ResponseWriter, statusCode int, body interface{}) {
	w.WriteHeader(statusCode)

	if body != nil {
		json.NewEncoder(w).Encode(body)
	}
}
