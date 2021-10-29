package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/code-cord/cc.core.server/handler/middleware"
	"github.com/code-cord/cc.core.server/service"
	"github.com/gorilla/mux"
)

func (h *Router) storageBackup(w http.ResponseWriter, r *http.Request) {
	storageName := mux.Vars(r)["name"]
	fileName := fmt.Sprintf("backup_%s_%s.db", storageName, time.Now().UTC().Format(time.RFC3339))

	w.Header().Set("Content-Disposition", "attachment; filename="+fileName)
	w.Header().Set("Content-Type", "application/octet-stream")

	if err := h.server.StorageBackup(
		r.Context(), service.ServerStorage(storageName), w); err != nil {
		middleware.WriteJSONResponse(w, http.StatusInternalServerError,
			middleware.ErrBackupStorage.New(err.Error()))
	}
}
