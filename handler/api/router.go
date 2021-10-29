package api

import (
	"net/http"

	"github.com/code-cord/cc.core.server/service"
	"github.com/gorilla/mux"
)

// Router represents server api router implementation model.
type Router struct {
	*mux.Router
	server service.Server
}

// Config represents router configuration model.
type Config struct {
	Server service.Server
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

	r.Path("/ping").
		HandlerFunc(r.ping)

	r.Path("/token").
		Methods(http.MethodPost).
		HandlerFunc(r.generateToken)

	r.Path("/stream").
		Methods(http.MethodGet).
		HandlerFunc(r.getStreams)

	r.Path("/stream/{uuid}").
		Methods(http.MethodDelete).
		HandlerFunc(r.finishStream)

	r.Path("/storage/{name}").
		Methods(http.MethodGet).
		HandlerFunc(r.storageBackup)

	return r
}
