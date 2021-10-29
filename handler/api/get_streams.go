package api

import (
	"net/http"

	"github.com/code-cord/cc.core.server/handler/middleware"
	"github.com/code-cord/cc.core.server/handler/models"
	"github.com/code-cord/cc.core.server/service"
)

func (h *Router) getStreams(w http.ResponseWriter, r *http.Request) {
	var req models.StreamListRequest
	if err := middleware.ParseURLRequest(r, &req); err != nil {
		middleware.WriteJSONResponse(w, http.StatusBadRequest, err)
		return
	}

	streams, err := h.server.StreamList(r.Context(), service.StreamFilter{
		SearchPhrase: req.Term,
		LaunchModes:  req.LaunchModes,
		Statuses:     req.Statuses,
		SortBy:       req.SortBy,
		SortOrder:    req.SortOrder,
		PageSize:     req.PageSize,
		Page:         req.Page,
	})
	if err != nil {
		middleware.WriteJSONResponse(w, http.StatusInternalServerError,
			middleware.ErrStreamList.New(err.Error()))
		return
	}

	middleware.WriteJSONResponse(w, http.StatusOK, buildStreamListResponse(streams))
}

func buildStreamListResponse(streams *service.StreamList) models.StreamListResponse {
	resp := models.StreamListResponse{
		Streams:  make([]models.StreamInfoResponse, len(streams.Streams)),
		Page:     streams.Page,
		PageSize: streams.PageSize,
		Count:    streams.Count,
		HasNext:  streams.HasNext,
		Total:    streams.Total,
	}

	for i := range streams.Streams {
		stream := &streams.Streams[i]

		resp.Streams[i] = models.StreamInfoResponse{
			UUID:        stream.UUID,
			Name:        stream.Name,
			Description: stream.Description,
			IP:          stream.IP,
			Port:        stream.Port,
			LaunchMode:  stream.LaunchMode,
			StartedAt:   stream.StartedAt,
			FinishedAt:  stream.FinishedAt,
			Status:      stream.Status,
			Join: models.StreamJoinConfigResponse{
				JoinPolicy: stream.Join.JoinPolicy,
				JoinCode:   stream.Join.JoinCode,
			},
			Host: models.HostOwnerInfo{
				UUID:     stream.Host.UUID,
				Username: stream.Host.Username,
				AvatarID: stream.Host.AvatarID,
				IP:       stream.Host.IP,
			},
		}
	}

	return resp
}
