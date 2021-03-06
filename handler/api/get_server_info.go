package api

import (
	"net/http"

	"github.com/code-cord/cc.core.server/handler/middleware"
	"github.com/code-cord/cc.core.server/handler/models"
	"github.com/code-cord/cc.core.server/service"
)

func (h *Router) getServerInfo(w http.ResponseWriter, r *http.Request) {
	resp := buildServerInfoResponse(h.server)

	middleware.WriteJSONResponse(w, http.StatusOK, resp)
}

func buildServerInfoResponse(s service.Server) models.ServerInfoResponse {
	info := s.Info()

	return models.ServerInfoResponse{
		Name:           info.Name,
		Description:    info.Description,
		Version:        info.Version,
		AdditionalInfo: info.Meta,
	}
}
