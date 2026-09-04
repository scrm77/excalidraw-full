package documents

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"excalidraw-complete/stores/memory"
)

func TestHandleCreateRejectsOversizedDocument(t *testing.T) {
	handler := HandleCreate(memory.NewStore())
	req := httptest.NewRequest(http.MethodPost, "/api/v2/post/", strings.NewReader(strings.Repeat("x", maxShareDocumentBytes+1)))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, req)

	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("POST status = %d, want %d; body=%s", response.Code, http.StatusRequestEntityTooLarge, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"error_class":"RequestTooLargeError"`) {
		t.Fatalf("unexpected response body: %s", response.Body.String())
	}
}

func TestHandleCreateRateLimitsPerClient(t *testing.T) {
	limiter := newShareCreateLimiter()
	limiter.perClient = 2
	limiter.global = 10
	limiter.now = func() time.Time { return time.Unix(1_000, 0) }
	handler := handleCreate(memory.NewStore(), limiter)

	for attempt, want := range []int{http.StatusOK, http.StatusOK, http.StatusTooManyRequests} {
		req := httptest.NewRequest(http.MethodPost, "/api/v2/post/", strings.NewReader("encrypted"))
		req.Header.Set("X-Forwarded-For", "198.51.100.10")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, req)
		if response.Code != want {
			t.Fatalf("attempt %d status = %d, want %d; body=%s", attempt+1, response.Code, want, response.Body.String())
		}
	}
}

func TestClientIPUsesLastForwardedAddress(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v2/post/", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.20, 198.51.100.30")

	if got, want := clientIP(req), "198.51.100.30"; got != want {
		t.Fatalf("clientIP() = %q, want %q", got, want)
	}
}
