package api

import "context"

// Server describes server API.
type Server interface {
	Info() ServerInfo
	Ping(ctx context.Context) error
}

// ServerInfo represents server info model.
type ServerInfo struct {
	Name        string
	Description string
	Version     string
	Meta        map[string]interface{}
}
