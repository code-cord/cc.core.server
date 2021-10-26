package handler

import (
	"net/http"

	"github.com/code-cord/cc.core.server/handler/middleware"
	"github.com/code-cord/cc.core.server/handler/models"
	"github.com/gorilla/mux"
)

func (h *Router) newAuthToken(w http.ResponseWriter, r *http.Request) {
	var subject string
	if v := r.Context().Value(middleware.ServerSubjectKey); v != nil {
		subject = v.(string)
	}
	streamUUID := mux.Vars(r)["uuid"]

	authInfo, err := h.server.NewStreamHostToken(r.Context(), streamUUID, subject)
	if err != nil {
		middleware.WriteJSONResponse(w, http.StatusInternalServerError,
			middleware.ErrGenerateAccessToken.New(err.Error()))
		return
	}

	resp := models.AuthorizationInfo{
		AccessToken: authInfo.AccessToken,
		Type:        authInfo.Type,
	}

	middleware.WriteJSONResponse(w, http.StatusCreated, resp)
}
