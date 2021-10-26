package handler

import (
	"net/http"

	"github.com/code-cord/cc.core.server/api"
	"github.com/gorilla/mux"
)

// Router represents server router implementation model.
type Router struct {
	*mux.Router
	server api.Server
	avatar api.Avatar
}

// Config represents router configuration model.
type Config struct {
	Server api.Server
	Avatar api.Avatar
}

// New returns new Router instance.
func New(cfg Config) Router {
	r := Router{
		Router: mux.NewRouter(),
		server: cfg.Server,
		avatar: cfg.Avatar,
	}

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

	r.Path("/stream").
		Methods(http.MethodPost).
		HandlerFunc(r.createStream)
	r.Path("/stream/{uuid}").
		Methods(http.MethodGet).
		HandlerFunc(r.getStreamInfo)
	r.Path("/stream/{uuid}/join").
		Methods(http.MethodPost).
		HandlerFunc(r.joinStream)
	r.Path("/stream/{uuid}/participants").
		Methods(http.MethodGet).
		HandlerFunc(r.getStreamParticipants)
	r.Path("/stream/{uuid}/participants/{participantUUID}/decision").
		Methods(http.MethodGet).
		HandlerFunc(r.joinParticipantDecision)

	return r
}
