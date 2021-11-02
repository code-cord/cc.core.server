package handler

import (
	"net/http"

	"github.com/code-cord/cc.core.server/handler/middleware"
	"github.com/code-cord/cc.core.server/handler/models"
	"github.com/code-cord/cc.core.server/service"
	"github.com/gorilla/mux"
)

func (h *Router) patchStream(w http.ResponseWriter, r *http.Request) {
	var req models.PatchStreamRequest
	if err := middleware.ParseJSONRequest(r, &req); err != nil {
		middleware.WriteJSONResponse(w, http.StatusBadRequest, err)
		return
	}

	streamUUID := mux.Vars(r)["uuid"]

	cfg := service.PatchStreamConfig{
		Name:        req.Name,
		Description: req.Description,
	}

	if req.Join != nil {
		cfg.Join = &service.StreamJoinPolicyConfig{
			JoinPolicy: req.Join.Policy,
			JoinCode:   req.Join.Code,
		}
	}
	if cfg.Host != nil {
		cfg.Host = &service.StreamHostConfig{
			Username: req.Host.Name,
			AvatarID: req.Host.AvatarID,
		}
	}

	streamInfo, err := h.server.PatchStream(r.Context(), streamUUID, cfg)
	if err != nil {
		middleware.WriteJSONResponse(w, http.StatusInternalServerError,
			middleware.ErrUpdateStream.New(err.Error()))
		return
	}

	resp := buildStreamOwnerInfoResponse(streamInfo)
	middleware.WriteJSONResponse(w, http.StatusOK, resp)
}
