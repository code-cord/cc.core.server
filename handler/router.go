package handler

import (
	"crypto/rsa"
	"net/http"

	"github.com/code-cord/cc.core.server/api"
	"github.com/code-cord/cc.core.server/handler/middleware"
	"github.com/gorilla/mux"
)

// Router represents server router implementation model.
type Router struct {
	*mux.Router
	server api.Server
}

// Config represents router configuration model.
type Config struct {
	Server               api.Server
	SeverSecurityEnabled bool
	ServerPublicKey      *rsa.PublicKey
}

// New returns new Router instance.
func New(cfg Config) Router {
	r := Router{
		Router: mux.NewRouter(),
		server: cfg.Server,
	}

	// public endpoints.
	r.Path("/").
		Methods(http.MethodGet).
		HandlerFunc(r.getServerInfo)

	r.Path("/ping").
		HandlerFunc(r.ping)

	r.Path("/avatar").
		Methods(http.MethodPost).
		HandlerFunc(r.addAvatar)
	r.Path("/avatar/{id}").
		Methods(http.MethodGet).
		HandlerFunc(r.getAvatar)

	r.Path("/stream/{uuid}").
		Methods(http.MethodGet).
		HandlerFunc(r.getStreamInfo)
	r.Path("/stream/{uuid}/join").
		Methods(http.MethodPost).
		HandlerFunc(r.joinStream)

	// server secure endpoints.
	serverSecureRouter := r.NewRoute().Subrouter()
	serverSecureRouter.Path("/stream").Subrouter().
		Methods(http.MethodPost).
		HandlerFunc(r.createStream)
	if cfg.SeverSecurityEnabled {
		serverSecureRouter.Path("/stream/{uuid}/token").
			Methods(http.MethodGet).
			HandlerFunc(r.newAuthToken)

		serverSecureRouter.Use(middleware.ServerAuthMiddleware(cfg.ServerPublicKey))
	}

	// stream secure endpoints.
	streamSecureRouter := r.NewRoute().Subrouter()
	streamSecureRouter.Use(middleware.StreamAuthMiddleware(cfg.Server, false))
	streamSecureRouter.Path("/stream/{uuid}/participants").
		Methods(http.MethodGet).
		HandlerFunc(r.getStreamParticipants)

	streamSecureHostRouter := r.NewRoute().Subrouter()
	streamSecureHostRouter.Use(middleware.StreamAuthMiddleware(cfg.Server, true))
	streamSecureHostRouter.Path("/stream/{uuid}/participants/{participantUUID}/decision").
		Methods(http.MethodGet).
		HandlerFunc(r.joinParticipantDecision)
	streamSecureHostRouter.Path("/stream/{uuid}").
		Methods(http.MethodDelete).
		HandlerFunc(r.finishStream)
	streamSecureHostRouter.Path("/stream/{uuid}").
		Methods(http.MethodPatch).
		HandlerFunc(r.patchStream)

	return r
}
