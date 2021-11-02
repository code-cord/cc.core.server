package handler

import (
	"fmt"
	"net/http"

	"github.com/code-cord/cc.core.server/handler/middleware"
	"github.com/gorilla/mux"
)

func (h *Router) streamProxy(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	streamUUID := vars["uuid"]

	streamAddress, err := h.server.StreamAddress(r.Context(), streamUUID)
	if err != nil {
		middleware.WriteJSONResponse(w, http.StatusInternalServerError,
			middleware.ErrStreamInfo.New(err.Error()))
		return
	}

	route := vars["route"]
	redirectURL := fmt.Sprintf("http://%s/%s", streamAddress, route)

	http.Redirect(w, r, redirectURL, http.StatusPermanentRedirect)
}
