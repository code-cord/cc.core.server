package handler

import (
	"net/http"

	"github.com/code-cord/cc.core.server/handler/middleware"
	"github.com/code-cord/cc.core.server/handler/models"
	"github.com/code-cord/cc.core.server/service"
	"github.com/gorilla/mux"
)

func (h *Router) getStreamParticipants(w http.ResponseWriter, r *http.Request) {
	streamUUID := mux.Vars(r)["uuid"]
	participants, err := h.server.StreamParticipants(r.Context(), streamUUID)
	if err != nil {
		middleware.WriteJSONResponse(w, http.StatusInternalServerError,
			middleware.ErrFetchStreamParticipants.New(err.Error()))
		return
	}

	resp := buildStreamParticipantsResponse(participants)

	middleware.WriteJSONResponse(w, http.StatusOK, resp)
}

func buildStreamParticipantsResponse(
	participants []service.Participant) []models.ParticipantResponse {
	resp := make([]models.ParticipantResponse, len(participants))

	for i := range participants {
		p := &participants[i]

		resp[i] = buildStreamParticipantResponse(p)
	}

	return resp
}

func buildStreamParticipantResponse(participant *service.Participant) models.ParticipantResponse {
	return models.ParticipantResponse{
		UUID:     participant.UUID,
		Name:     participant.Name,
		AvatarID: participant.AvatarID,
		Status:   participant.Status,
	}
}
