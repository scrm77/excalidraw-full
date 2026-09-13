package auth

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"golang.org/x/oauth2"
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

func TestOIDCLoginFlow(t *testing.T) {
	original := oidcOauthConfig
	t.Cleanup(func() { oidcOauthConfig = original })
	oidcOauthConfig = &oauth2.Config{
		ClientID: "draw", RedirectURL: "https://draw.example/auth/callback",
		Scopes:   []string{"openid", "profile"},
		Endpoint: oauth2.Endpoint{AuthURL: "https://auth.example/application/o/authorize/"},
	}
	for _, tt := range []struct {
		name, flow string
		wantError  bool
	}{
		{name: "normal OIDC"},
		{name: "explicit personal login", flow: "https://auth.example/if/flow/personal/"},
		{name: "reject foreign host", flow: "https://evil.example/login", wantError: true},
		{name: "reject cleartext", flow: "http://auth.example/login", wantError: true},
		{name: "reject relative URL", flow: "/login", wantError: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("OIDC_LOGIN_FLOW_URL", tt.flow)
			r := httptest.NewRequest("GET", "/auth/login", nil)
			w := httptest.NewRecorder()
			HandleOIDCLogin(w, r)
			if tt.wantError {
				if w.Code != http.StatusInternalServerError {
					t.Fatalf("status = %d", w.Code)
				}
				return
			}
			if w.Code != http.StatusTemporaryRedirect {
				t.Fatalf("status = %d", w.Code)
			}
			target, err := url.Parse(w.Header().Get("Location"))
			if err != nil {
				t.Fatal(err)
			}
			if tt.flow != "" {
				if target.Path != "/if/flow/personal/" {
					t.Fatalf("unexpected flow: %s", target.Path)
				}
				target, err = url.Parse(target.Query().Get("next"))
				if err != nil {
					t.Fatal(err)
				}
			}
			q := target.Query()
			if target.Path != "/application/o/authorize/" || q.Get("client_id") != "draw" || q.Get("redirect_uri") != oidcOauthConfig.RedirectURL {
				t.Fatal("OIDC destination/parameters not preserved")
			}
			cookies := w.Result().Cookies()
			if len(cookies) != 1 || cookies[0].Name != oidcStateCookie || cookies[0].Value == "" || q.Get("state") != cookies[0].Value {
				t.Fatal("state/cookie binding not preserved")
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
