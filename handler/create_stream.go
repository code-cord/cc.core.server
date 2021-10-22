package handler

import (
	"net/http"

	"github.com/code-cord/cc.core.server/handler/middleware"
)

func (h *Router) createStream(w http.ResponseWriter, r *http.Request) {
	resp := buildServerInfoResponse(h.server)

	middleware.WriteJSONResponse(w, http.StatusOK, resp)
}
