package handler

import (
	"net/http"

	"github.com/code-cord/cc.core.server/api"
	"github.com/code-cord/cc.core.server/handler/middleware"
	"github.com/code-cord/cc.core.server/handler/models"
	"github.com/gorilla/mux"
)

func (h *Router) getStreamInfo(w http.ResponseWriter, r *http.Request) {
	streamUUID := mux.Vars(r)["uuid"]
	streamInfo := h.server.StreamInfo(r.Context(), streamUUID)
	if streamInfo == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	resp := buildStreamInfoResponse(streamInfo)

	middleware.WriteJSONResponse(w, http.StatusOK, resp)
}

func buildStreamInfoResponse(info *api.StreamPublicInfo) models.StreamPublicInfoResponse {
	return models.StreamPublicInfoResponse{
		UUID:        info.UUID,
		Name:        info.Name,
		Description: info.Description,
		JoinPolicy:  info.JoinPolicy,
		StartedAt:   info.StartedAt,
	}
}
