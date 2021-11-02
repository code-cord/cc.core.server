package handler

import (
	"net/http"

	"github.com/code-cord/cc.core.server/handler/middleware"
	"github.com/code-cord/cc.core.server/handler/models"
	"github.com/code-cord/cc.core.server/service"
	"github.com/gorilla/mux"
)

func (h *Router) patchParticipant(w http.ResponseWriter, r *http.Request) {
	ctxData := r.Context().Value(middleware.ParticipantKey)
	if ctxData == nil {
		middleware.WriteJSONResponse(w, http.StatusUnauthorized,
			middleware.ErrAuth.New("invalid context data"))
		return
	}
	participant := ctxData.(middleware.ParticipantCtxData)

	var req models.PatchParticipantRequest
	if err := middleware.ParseJSONRequest(r, &req); err != nil {
		middleware.WriteJSONResponse(w, http.StatusBadRequest, err)
		return
	}

	streamUUID := mux.Vars(r)["uuid"]

	cfg := service.PatchParticipantConfig{
		Name:     req.Name,
		AvatarID: req.AvatarID,
	}

	participantInfo, err := h.server.PatchParticipant(
		r.Context(), streamUUID, participant.UUID, cfg)
	if err != nil {
		middleware.WriteJSONResponse(w, http.StatusInternalServerError,
			middleware.ErrUpdateParticipantInfo.New(err.Error()))
		return
	}

	resp := buildStreamParticipantResponse(participantInfo)
	middleware.WriteJSONResponse(w, http.StatusOK, resp)
}
