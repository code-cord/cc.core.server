package api

import (
	"crypto/rsa"
	"net/http"

	"github.com/code-cord/cc.core.server/api"
	"github.com/gorilla/mux"
)

// Router represents server api router implementation model.
type Router struct {
	*mux.Router
	server     api.Server
	privateKey *rsa.PrivateKey
}

// Config represents router configuration model.
type Config struct {
	Server           api.Server
	ServerPrivateKey *rsa.PrivateKey
}

// New returns new router instance.
func New(cfg Config) Router {
	r := Router{
		Router:     mux.NewRouter(),
		server:     cfg.Server,
		privateKey: cfg.ServerPrivateKey,
	}

	r.Path("/").
		Methods(http.MethodGet).
		HandlerFunc(r.getServerInfo)

	r.Path("/token").
		Methods(http.MethodPost).
		HandlerFunc(r.generateToken)

	return r
}
