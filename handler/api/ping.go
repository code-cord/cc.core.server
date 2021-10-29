package api

import (
	"net/http"

	"github.com/code-cord/cc.core.server/handler/middleware"
	"github.com/code-cord/cc.core.server/handler/models"
)

const (
	defaultServerPongMessage = "pong"
)

func (h *Router) ping(w http.ResponseWriter, r *http.Request) {
	if err := h.server.Ping(r.Context()); err != nil {
		middleware.WriteJSONResponse(w,
			http.StatusInternalServerError, middleware.ErrServerPing.New(err.Error()))
		return
	}

	middleware.WriteJSONResponse(w, http.StatusOK, models.PongResponse{
		Message: defaultServerPongMessage,
	})
}
