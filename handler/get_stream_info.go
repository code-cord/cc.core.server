package handler

import (
	"net/http"
	"os"

	"github.com/code-cord/cc.core.server/handler/middleware"
	"github.com/code-cord/cc.core.server/handler/models"
	"github.com/code-cord/cc.core.server/service"
	"github.com/gorilla/mux"
)

func (h *Router) getStreamInfo(w http.ResponseWriter, r *http.Request) {
	streamUUID := mux.Vars(r)["uuid"]
	streamInfo, err := h.server.StreamInfo(r.Context(), streamUUID)
	if err != nil {
		status := http.StatusInternalServerError
		if os.IsNotExist(err) {
			status = http.StatusNotFound
		}

		middleware.WriteJSONResponse(w, status, middleware.ErrStreamInfo.New(err.Error()))
		return
	}

	resp := buildStreamInfoResponse(streamInfo)
	middleware.WriteJSONResponse(w, http.StatusOK, resp)
}

func buildStreamInfoResponse(info *service.StreamPublicInfo) models.StreamPublicInfoResponse {
	return models.StreamPublicInfoResponse{
		UUID:        info.UUID,
		Name:        info.Name,
		Description: info.Description,
		JoinPolicy:  info.JoinPolicy,
		StartedAt:   info.StartedAt,
		FinishedAt:  info.FinishedAt,
	}
}
