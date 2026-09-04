package documents

import (
	"encoding/json"
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

func TestShareLimitNotifierSendsAtMostOncePerCooldown(t *testing.T) {
	type alertRequest struct {
		authorization string
		body          map[string]string
	}
	received := make(chan alertRequest, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := map[string]string{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode alert body: %v", err)
		}
		received <- alertRequest{authorization: r.Header.Get("Authorization"), body: body}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	now := time.Unix(1_000, 0)
	notifier := &shareLimitNotifier{
		now:      func() time.Time { return now },
		cooldown: time.Hour,
		url:      server.URL,
		token:    "test-token",
		client:   server.Client(),
	}
	notifier.Notify("198.51.100.10")
	notifier.Notify("198.51.100.11")

	select {
	case request := <-received:
		if got, want := request.authorization, "Bearer test-token"; got != want {
			t.Fatalf("Authorization = %q, want %q", got, want)
		}
		if got, want := request.body["unit"], "anonymous-share-rate-limit"; got != want {
			t.Fatalf("unit = %q, want %q", got, want)
		}
	case <-time.After(time.Second):
		t.Fatal("alert was not sent")
	}

	select {
	case <-received:
		t.Fatal("second alert was sent during cooldown")
	case <-time.After(100 * time.Millisecond):
	}
}
