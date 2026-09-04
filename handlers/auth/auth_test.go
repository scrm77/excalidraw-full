package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestValidateOAuthState(t *testing.T) {
	tests := []struct {
		name        string
		queryState  string
		cookieState string
		want        bool
	}{
		{name: "matching state", queryState: "known-state", cookieState: "known-state", want: true},
		{name: "mismatched state", queryState: "attacker", cookieState: "known-state", want: false},
		{name: "missing query state", cookieState: "known-state", want: false},
		{name: "missing cookie state", queryState: "known-state", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/auth/callback?state="+tt.queryState, nil)
			if tt.cookieState != "" {
				r.AddCookie(&http.Cookie{Name: oidcStateCookie, Value: tt.cookieState})
			}
			if got := validateOAuthState(r, oidcStateCookie); got != tt.want {
				t.Fatalf("validateOAuthState() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJWTTTL(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want time.Duration
	}{
		{name: "default", want: defaultJWTTTL},
		{name: "configured", raw: "24h", want: 24 * time.Hour},
		{name: "invalid", raw: "tomorrow", want: defaultJWTTTL},
		{name: "non-positive", raw: "0s", want: defaultJWTTTL},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("JWT_TTL", tt.raw)
			if got := jwtTTL(); got != tt.want {
				t.Fatalf("jwtTTL() = %s, want %s", got, tt.want)
			}
		})
	}
}
