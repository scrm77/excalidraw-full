package middleware

import (
	"context"
	"crypto/subtle"
	"excalidraw-complete/handlers/auth"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/render"
	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const ClaimsContextKey = contextKey("claims")

const authMethodContextKey = contextKey("auth-method")

const (
	authMethodJWT      = "jwt"
	authMethodOwnerAPI = "owner-api"
)

func AuthJWT(next http.Handler) http.Handler {
	return authenticate(next, false)
}

// AuthJWTOrOwnerAPI allows the regular short-lived login JWT and the owner's
// long-lived API token. The owner token is intentionally limited by route:
// callers must also use ForbidOwnerAPIDelete on destructive handlers.
func AuthJWTOrOwnerAPI(next http.Handler) http.Handler {
	return authenticate(next, true)
}

func authenticate(next http.Handler, allowOwnerAPI bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, map[string]string{"error": "Authorization header is required"})
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, map[string]string{"error": "Authorization header format must be Bearer {token}"})
			return
		}

		tokenString := parts[1]
		if allowOwnerAPI && isOwnerAPIToken(tokenString) {
			userID := strings.TrimSpace(os.Getenv("DRAW_MEATBAGS_OWNER_API_USER_ID"))
			if userID == "" {
				render.Status(r, http.StatusUnauthorized)
				render.JSON(w, r, map[string]string{"error": "Invalid token"})
				return
			}

			claims := &auth.AppClaims{
				RegisteredClaims: jwt.RegisteredClaims{Subject: userID},
				Login:            "owner-api",
			}
			ctx := context.WithValue(r.Context(), ClaimsContextKey, claims)
			ctx = context.WithValue(ctx, authMethodContextKey, authMethodOwnerAPI)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		claims, err := auth.ParseJWT(tokenString)
		if err != nil {
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, map[string]string{"error": "Invalid token"})
			return
		}

		ctx := context.WithValue(r.Context(), ClaimsContextKey, claims)
		ctx = context.WithValue(ctx, authMethodContextKey, authMethodJWT)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func isOwnerAPIToken(token string) bool {
	configured := strings.TrimSpace(os.Getenv("DRAW_MEATBAGS_OWNER_API_TOKEN"))
	if configured == "" || token == "" || len(configured) != len(token) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(configured), []byte(token)) == 1
}

// ForbidOwnerAPIDelete keeps the personal automation token non-destructive.
// Interactive users authenticated through GitHub retain the existing delete
// behavior.
func ForbidOwnerAPIDelete(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Context().Value(authMethodContextKey) == authMethodOwnerAPI {
			render.Status(r, http.StatusForbidden)
			render.JSON(w, r, map[string]string{"error": "Owner API token cannot delete canvases"})
			return
		}
		next.ServeHTTP(w, r)
	})
}
