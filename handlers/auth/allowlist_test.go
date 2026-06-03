package auth

import (
	"os"
	"testing"
)

func TestIsGitHubUserAllowed(t *testing.T) {
	cases := []struct {
		name, env, login string
		id               int64
		want             bool
	}{
		{"empty env = open (upstream behaviour)", "", "anybody", 999, true},
		{"owner allowed by login", "scrm77,5765513", "scrm77", 5765513, true},
		{"owner allowed by id even if login changed", "scrm77,5765513", "some-new-name", 5765513, true},
		{"login match is case-insensitive", "scrm77", "ScRm77", 5765513, true},
		{"spaces around entries tolerated", " scrm77 , 5765513 ", "scrm77", 5765513, true},
		{"DIFFERENT github account -> DENIED", "scrm77,5765513", "randomhacker", 42, false},
		{"another denied account", "scrm77,5765513", "octocat", 583231, false},
		{"empty-ish list denies", " , ", "x", 1, false},
	}
	for _, c := range cases {
		os.Setenv("ALLOWED_GITHUB_USERS", c.env)
		got := isGitHubUserAllowed(c.login, c.id)
		status := "OK"
		if got != c.want {
			status = "FAIL"
			t.Errorf("%s: got %v want %v", c.name, got, c.want)
		}
		t.Logf("[%s] %-45s login=%-14q id=%-8d allowed=%v", status, c.name, c.login, c.id, got)
	}
}
