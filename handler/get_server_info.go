package handler

import (
	"net/http"

	"github.com/code-cord/cc.core.server/api"
	"github.com/code-cord/cc.core.server/handler/middleware"
	"github.com/code-cord/cc.core.server/handler/models"
)

func (h *Router) getServerInfo(w http.ResponseWriter, r *http.Request) {
	resp := buildServerInfoResponse(h.server)

	middleware.WriteJSONResponse(w, http.StatusOK, resp)
}

func buildServerInfoResponse(s api.Server) models.ServerInfoResponse {
	info := s.Info()

	return models.ServerInfoResponse{
		Name:           info.Name,
		Description:    info.Description,
		Version:        info.Version,
		AdditionalInfo: info.Meta,
	}
}
