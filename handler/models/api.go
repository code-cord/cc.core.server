package models

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
)

// GenerateServerTokenRequest represents generate server token request model.
type GenerateServerTokenRequest struct {
	Audience  string    `json:"aud,omitempty"`
	ExpiresAt time.Time `json:"exp,omitempty"`
	IssuedAt  time.Time `json:"iat,omitempty"`
	Issuer    string    `json:"iss,omitempty"`
	NotBefore time.Time `json:"nbf,omitempty"`
	Subject   string    `json:"sub"`
}

// Validate validates request model.
func (req *GenerateServerTokenRequest) Validate() error {
	return validation.Errors{
		"sub": validation.Validate(req.Subject,
			validation.Required,
			validation.Length(10, 64),
		),
	}.Filter()
}
