package handler

import (
	"net/http"
	"os"

	"github.com/code-cord/cc.core.server/handler/middleware"
	"github.com/gorilla/mux"
)

func (h *Router) getAvatar(w http.ResponseWriter, r *http.Request) {
	avatarID := mux.Vars(r)["id"]

	imgData, contentType, err := h.server.AvatarByID(r.Context(), avatarID)
	if err != nil {
		if os.IsNotExist(err) {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		middleware.WriteJSONResponse(w, http.StatusBadRequest,
			middleware.ErrInvalidRequest.New(err.Error()))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", contentType)
	w.Write(imgData)
}
