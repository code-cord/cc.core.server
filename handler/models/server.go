package models

// ServerInfoResponse represents server info response model.
type ServerInfoResponse struct {
	Name           string                 `json:"name"`
	Description    string                 `json:"description"`
	Version        string                 `json:"version"`
	AdditionalInfo map[string]interface{} `json:"info,omitempty"`
}

// PongResponse represents pong response model.
type PongResponse struct {
	Message string `json:"message"`
}
