package middleware

import (
	"excalidraw-complete/handlers/auth"
	"net/http"
	"net/http/httptest"
	"testing"
)

const testOwnerToken = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func TestOwnerAPIAuthentication(t *testing.T) {
	t.Setenv("DRAW_MEATBAGS_OWNER_API_TOKEN", testOwnerToken)
	t.Setenv("DRAW_MEATBAGS_OWNER_API_USER_ID", "github:5765513")

	handler := AuthJWTOrOwnerAPI(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := r.Context().Value(ClaimsContextKey).(*auth.AppClaims)
		if !ok {
			t.Fatal("owner claims missing from request context")
		}
		if claims.Subject != "github:5765513" {
			t.Fatalf("subject = %q, want github:5765513", claims.Subject)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v2/kv/", nil)
	req.Header.Set("Authorization", "Bearer "+testOwnerToken)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusNoContent)
	}
}

func TestOwnerAPIAuthenticationRequiresConfiguredUser(t *testing.T) {
	t.Setenv("DRAW_MEATBAGS_OWNER_API_TOKEN", testOwnerToken)
	t.Setenv("DRAW_MEATBAGS_OWNER_API_USER_ID", "")

	handler := AuthJWTOrOwnerAPI(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("request reached protected handler")
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/v2/kv/", nil)
	req.Header.Set("Authorization", "Bearer "+testOwnerToken)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusUnauthorized)
	}
}

func TestOwnerAPICannotDelete(t *testing.T) {
	t.Setenv("DRAW_MEATBAGS_OWNER_API_TOKEN", testOwnerToken)
	t.Setenv("DRAW_MEATBAGS_OWNER_API_USER_ID", "github:5765513")

	handler := AuthJWTOrOwnerAPI(ForbidOwnerAPIDelete(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("owner API delete reached protected handler")
	})))
	req := httptest.NewRequest(http.MethodDelete, "/api/v2/kv/test/", nil)
	req.Header.Set("Authorization", "Bearer "+testOwnerToken)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusForbidden)
	}
}

func TestOwnerTokenIsNotAcceptedByJWTOnlyRoutes(t *testing.T) {
	t.Setenv("DRAW_MEATBAGS_OWNER_API_TOKEN", testOwnerToken)
	t.Setenv("DRAW_MEATBAGS_OWNER_API_USER_ID", "github:5765513")

	handler := AuthJWT(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("owner token reached JWT-only handler")
	}))
	req := httptest.NewRequest(http.MethodPost, "/api/v2/chat/completions", nil)
	req.Header.Set("Authorization", "Bearer "+testOwnerToken)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusUnauthorized)
	}
}
