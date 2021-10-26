package handler

import (
	"net/http"

	"github.com/code-cord/cc.core.server/handler/middleware"
	"github.com/gorilla/mux"
)

func (h *Router) joinParticipantDecision(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	streamUUID := vars["uuid"]
	participantUUID := vars["participantUUID"]
	allowed := r.URL.Query().Has("allowed")

	err := h.server.DecideParticipantJoin(r.Context(), streamUUID, participantUUID, allowed)
	if err != nil {
		middleware.WriteJSONResponse(w, http.StatusInternalServerError,
			middleware.ErrDecideParticipantJoin.New(err.Error()))
		return
	}

	middleware.WriteJSONResponse(w, http.StatusOK, nil)
}
