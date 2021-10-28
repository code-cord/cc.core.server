package api

import (
	"net/http"

	"github.com/code-cord/cc.core.server/handler/middleware"
	"github.com/code-cord/cc.core.server/handler/models"
)

func (h *Router) generateToken(w http.ResponseWriter, r *http.Request) {
	var req models.GenerateServerTokenRequest
	if err := middleware.ParseJSONRequest(r, &req); err != nil {
		middleware.WriteJSONResponse(w, http.StatusBadRequest, err)
		return
	}

	/*middleware.UpgradeRequestToSSE(w, "*")
	flusher, ok := w.(http.Flusher)
	if !ok {
		middleware.WriteJSONResponse(w, http.StatusBadRequest, middleware.ErrSSEUpgrade.New(nil))
		return
	}
	defer flusher.Flush()

	streamUUID := mux.Vars(r)["uuid"]

	joinDecision, err := h.server.JoinParticipant(
		r.Context(), streamUUID, req.JoinCode, api.Participant{
			Name:     req.Name,
			AvatarID: req.AvatarID,
			IP:       util.GetIP(r),
		})
	if err != nil {
		middleware.WriteJSONResponse(w, http.StatusInternalServerError,
			middleware.ErrJoinStream.New(err.Error()))
		return
	}

	resp := models.ParticipantJoinResponse{
		Allowed:     joinDecision.JoinAllowed,
		AccessToken: joinDecision.AccessToken,
	}

	middleware.WriteJSONResponse(w, http.StatusOK, resp)*/
}
