package api

import (
	"net/http"

	"github.com/code-cord/cc.core.server/handler/middleware"
	"github.com/code-cord/cc.core.server/handler/models"
	"github.com/golang-jwt/jwt"
)

func (h *Router) generateToken(w http.ResponseWriter, r *http.Request) {
	var req models.GenerateServerTokenRequest
	if err := middleware.ParseJSONRequest(r, &req); err != nil {
		middleware.WriteJSONResponse(w, http.StatusBadRequest, err)
		return
	}

	token, err := h.server.NewServerToken(r.Context(), &jwt.StandardClaims{
		Audience:  req.Audience,
		ExpiresAt: req.ExpiresAt.Unix(),
		IssuedAt:  req.IssuedAt.Unix(),
		Issuer:    req.Issuer,
		NotBefore: req.NotBefore.Unix(),
		Subject:   req.Subject,
	})
	if err != nil {
		middleware.WriteJSONResponse(w, http.StatusInternalServerError,
			middleware.ErrGenerateToken.New(err.Error()))
		return
	}

	middleware.WriteJSONResponse(w, http.StatusCreated, models.ServerTokenResponse{
		AccessToken: token.AccessToken,
	})
}
