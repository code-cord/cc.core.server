package handler

import (
	"fmt"
	"net/http"

	"github.com/code-cord/cc.core.server/handler/middleware"
	"github.com/code-cord/cc.core.server/handler/models"
)

func (h *Router) addAvatar(w http.ResponseWriter, r *http.Request) {
	file, fileHeader, err := r.FormFile("avatar")
	if err != nil {
		middleware.WriteJSONResponse(w, http.StatusBadRequest,
			middleware.ErrInvalidRequest.New(err.Error()))
		return
	}

	restrictions := h.avatar.Restrictions()
	if restrictions.MaxFileSize != 0 && restrictions.MaxFileSize <= fileHeader.Size {
		middleware.WriteJSONResponse(w, http.StatusBadRequest,
			middleware.ErrInvalidRequest.New(
				fmt.Sprintf("file size should be less than %d bytes", restrictions.MaxFileSize)))
		return
	}

	contentTypeHeaders, ok := fileHeader.Header["Content-Type"]
	if !ok || len(contentTypeHeaders) == 0 {
		middleware.WriteJSONResponse(w, http.StatusBadRequest,
			middleware.ErrInvalidRequest.New("could not detect content type of the image"))
		return
	}

	avatarID, err := h.avatar.New(r.Context(), contentTypeHeaders[0], file)
	if err != nil {
		middleware.WriteJSONResponse(w, http.StatusBadRequest,
			middleware.ErrInvalidRequest.New(err.Error()))
		return
	}
	defer file.Close()

	middleware.WriteJSONResponse(w, http.StatusCreated, models.AddAvatarResponse{
		AvatarID: avatarID,
	})
}
