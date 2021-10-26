package handler

import (
	"net/http"

	"github.com/code-cord/cc.core.server/handler/middleware"
	"github.com/gorilla/mux"
)

func (h *Router) finishStream(w http.ResponseWriter, r *http.Request) {
	streamUUID := mux.Vars(r)["uuid"]

	if err := h.server.FinishStream(r.Context(), streamUUID); err != nil {
		middleware.WriteJSONResponse(w, http.StatusInternalServerError,
			middleware.ErrFinishStream.New(err.Error()))
		return
	}

	middleware.WriteJSONResponse(w, http.StatusOK, nil)
}
