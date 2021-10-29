package handler

import (
	"net/http"

	"github.com/code-cord/cc.core.server/handler/middleware"
	"github.com/code-cord/cc.core.server/handler/models"
	"github.com/code-cord/cc.core.server/service"
	"github.com/code-cord/cc.core.server/util"
)

func (h *Router) createStream(w http.ResponseWriter, r *http.Request) {
	var subject string
	if v := r.Context().Value(middleware.ServerSubjectKey); v != nil {
		subject = v.(string)
	}

	var req models.CreateStreamRequest
	if err := middleware.ParseJSONRequest(r, &req); err != nil {
		middleware.WriteJSONResponse(w, http.StatusBadRequest, err)
		return
	}

	streamInfo, err := h.server.NewStream(r.Context(), service.StreamConfig{
		Name:        req.Name,
		Description: req.Description,
		Join: service.StreamJoinPolicyConfig{
			JoinPolicy: req.Join.Policy,
			JoinCode:   req.Join.Code,
		},
		Launch: service.StreamLaunchConfig{
			PreferredPort: req.Stream.PreferredPort,
			PreferredIP:   req.Stream.PreferredIP,
			Mode:          req.Stream.LaunchMode,
		},
		Host: service.StreamHostConfig{
			Username: req.Host.Name,
			AvatarID: req.Host.AvatarID,
			IP:       util.GetIP(r),
		},
		Subject: subject,
	})
	if err != nil {
		middleware.WriteJSONResponse(w, http.StatusInternalServerError,
			middleware.ErrCreateStream.New(err.Error()))
		return
	}

	resp := buildStreamOwnerInfoResponse(streamInfo)

	middleware.WriteJSONResponse(w, http.StatusCreated, resp)
}

func buildStreamOwnerInfoResponse(info *service.StreamOwnerInfo) models.StreamOwnerInfoResponse {
	resp := models.StreamOwnerInfoResponse{
		UUID:        info.UUID,
		Name:        info.Name,
		Description: info.Description,
		JoinPolicy:  info.JoinPolicy,
		StartedAt:   info.StartedAt,
		Port:        info.Port,
		IP:          info.IP,
		LaunchMode:  info.LaunchMode,
		HostInfo: models.HostOwnerInfo{
			UUID:     info.Host.UUID,
			Username: info.Host.Username,
			AvatarID: info.Host.AvatarID,
			IP:       info.Host.IP,
		},
	}

	if info.JoinPolicy == service.JoinPolicyByCode {
		resp.JoinCode = info.JoinCode
	}

	if info.Auth != nil {
		resp.Auth = &models.AuthorizationInfo{
			AccessToken: info.Auth.AccessToken,
			Type:        info.Auth.Type,
		}
	}

	return resp
}
