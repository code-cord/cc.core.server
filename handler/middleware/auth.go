package middleware

import (
	"context"
	"crypto/rsa"
	"fmt"
	"net/http"

	"github.com/golang-jwt/jwt"
)

const (
	serverAuthTokenHeader = "X-CODE-CORD-AUTH"
)

// Server context key.
const (
	ServerSubjectKey ServerKey = "subject"
)

// ServerKey represents server subject context key.
type ServerKey string

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
