package middleware

import (
	"context"
	"crypto/rsa"
	"fmt"
	"net/http"
	"strings"

	"github.com/code-cord/cc.core.server/service"
	"github.com/golang-jwt/jwt"
	"github.com/gorilla/mux"
)

const (
	serverAuthTokenHeader = "X-CODE-CORD-AUTH"
	authTokenHeader       = "Authorization"
	bearerPrefix          = "Bearer "
)

// Server context key.
const (
	ServerSubjectKey ContextKey = "subject"
	ParticipantKey   ContextKey = "participant"
)

// ContextKey represents context key type.
type ContextKey string

// ParticipantCtxData represents participant context data.
type ParticipantCtxData struct {
	UUID       string
	StreamUUID string
	IsHost     bool
}

// ServerAuthMiddleware represents middleware func to check access to the server-side operations.
func ServerAuthMiddleware(publicKey *rsa.PublicKey) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authToken := r.Header.Get(serverAuthTokenHeader)

			token, err := jwt.Parse(authToken, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
					return nil, fmt.Errorf("unexpected method: %s", t.Header["alg"])
				}

				return publicKey, nil
			})
			if err != nil {
				WriteJSONResponse(w, http.StatusUnauthorized, ErrAuth.New(err.Error()))
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok || !token.Valid {
				WriteJSONResponse(w, http.StatusUnauthorized, ErrAuth.New("invalid token"))
				return
			}

			subject, ok := claims["sub"].(string)
			if !ok || subject == "" {
				WriteJSONResponse(w, http.StatusUnauthorized,
					ErrAuth.New("coukd not find subject of the token"))
				return
			}

			ctx := context.WithValue(r.Context(), ServerSubjectKey, subject)
			r = r.WithContext(ctx)

			h.ServeHTTP(w, r)
		})
	}
}

// StreamAuthMiddleware represents middleware func to check access to the stream operations.
func StreamAuthMiddleware(
	server service.Server, hostSpecific bool) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authToken := r.Header.Get(authTokenHeader)
			authToken = strings.TrimPrefix(authToken, bearerPrefix)
			authToken = strings.TrimPrefix(authToken, strings.ToLower(bearerPrefix))
			streamUUID := mux.Vars(r)["uuid"]

			publicKey, err := server.StreamKey(r.Context(), streamUUID)
			if err != nil {
				WriteJSONResponse(w, http.StatusUnauthorized, ErrAuth.New(err.Error()))
				return
			}

			token, err := jwt.Parse(authToken, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
					return nil, fmt.Errorf("unexpected method: %s", t.Header["alg"])
				}

				return publicKey, nil
			})
			if err != nil {
				WriteJSONResponse(w, http.StatusUnauthorized, ErrAuth.New(err.Error()))
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok || !token.Valid {
				WriteJSONResponse(w, http.StatusUnauthorized, ErrAuth.New("invalid token"))
				return
			}

			participant := ParticipantCtxData{
				UUID:       claims["UUID"].(string),
				StreamUUID: claims["streamUUID"].(string),
			}
			if isHost, ok := claims["host"]; ok {
				participant.IsHost = isHost.(bool)
			}

			if streamUUID != participant.StreamUUID {
				WriteJSONResponse(w, http.StatusForbidden,
					ErrAuth.New("access denied"))
				return
			}

			if hostSpecific && !participant.IsHost {
				WriteJSONResponse(w, http.StatusUnauthorized,
					ErrAuth.New("only host of the stream has access to this endpoint"))
				return
			}

			ctx := context.WithValue(r.Context(), ParticipantKey, participant)
			r = r.WithContext(ctx)

			h.ServeHTTP(w, r)
		})
	}
}
