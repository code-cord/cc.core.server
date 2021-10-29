package api

import (
	"net/http"

	"github.com/code-cord/cc.core.server/api"
	"github.com/gorilla/mux"
)

// Router represents server api router implementation model.
type Router struct {
	*mux.Router
	server api.Server
}

// Config represents router configuration model.
type Config struct {
	Server api.Server
}

// New returns new router instance.
func New(cfg Config) Router {
	r := Router{
		Router: mux.NewRouter(),
		server: cfg.Server,
	}

	r.Path("/").
		Methods(http.MethodGet).
		HandlerFunc(r.getServerInfo)

	r.Path("/token").
		Methods(http.MethodPost).
		HandlerFunc(r.generateToken)

	r.Path("/stream").
		Methods(http.MethodGet).
		HandlerFunc(r.getStreams)

	return r
}
