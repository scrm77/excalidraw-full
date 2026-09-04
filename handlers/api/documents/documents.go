package documents

import (
	"bytes"
	"encoding/json"
	"excalidraw-complete/core"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type (
	DocumentCreateResponse struct {
		ID string `json:"id"`
	}
	ErrorResponse struct {
		ErrorClass string `json:"error_class"`
		Message    string `json:"message"`
	}
	windowCount struct {
		StartedAt time.Time
		Count     int
	}
	shareCreateLimiter struct {
		mu        sync.Mutex
		now       func() time.Time
		window    time.Duration
		perClient int
		global    int
		clients   map[string]windowCount
		all       windowCount
	}
	shareLimitNotifier struct {
		mu       sync.Mutex
		now      func() time.Time
		lastSent time.Time
		sending  bool
		cooldown time.Duration
		url      string
		token    string
		client   *http.Client
	}
)

const (
	maxShareDocumentBytes = 5 << 20
	shareRateWindow       = time.Hour
	shareRatePerClient    = 20
	shareRateGlobal       = 100
	shareAlertCooldown    = time.Hour
)

func HandleCreate(documentStore core.DocumentStore) http.HandlerFunc {
	return handleCreateWithNotifier(documentStore, newShareCreateLimiter(), newShareLimitNotifier())
}

func handleCreate(documentStore core.DocumentStore, limiter *shareCreateLimiter) http.HandlerFunc {
	return handleCreateWithNotifier(documentStore, limiter, newShareLimitNotifier())
}

func handleCreateWithNotifier(documentStore core.DocumentStore, limiter *shareCreateLimiter, notifier *shareLimitNotifier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		client := clientIP(r)
		if !limiter.Allow(client) {
			notifier.Notify(client)
			w.Header().Set("Retry-After", "3600")
			render.Status(r, http.StatusTooManyRequests)
			render.JSON(w, r, ErrorResponse{
				ErrorClass: "RateLimitError",
				Message:    "Too many anonymous share links. Please try again later.",
			})
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, maxShareDocumentBytes)
		data := new(bytes.Buffer)
		_, err := io.Copy(data, r.Body)
		if err != nil {
			if _, ok := err.(*http.MaxBytesError); ok {
				render.Status(r, http.StatusRequestEntityTooLarge)
				render.JSON(w, r, ErrorResponse{
					ErrorClass: "RequestTooLargeError",
					Message:    "Share document is too large.",
				})
				return
			}
			http.Error(w, "Failed to copy", http.StatusInternalServerError)
			return
		}
		id, err := documentStore.Create(r.Context(), &core.Document{Data: *data})
		if err != nil {
			http.Error(w, "Failed to save", http.StatusInternalServerError)
			return
		}

		render.JSON(w, r, DocumentCreateResponse{ID: id})
		render.Status(r, http.StatusOK)
	}
}

func newShareLimitNotifier() *shareLimitNotifier {
	return &shareLimitNotifier{
		now:      time.Now,
		cooldown: shareAlertCooldown,
		url:      strings.TrimSpace(os.Getenv("OPS_ALERTS_URL")),
		token:    strings.TrimSpace(os.Getenv("OPS_ALERTS_TOKEN_V2")),
		client:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (n *shareLimitNotifier) Notify(clientIP string) {
	if n.url == "" || n.token == "" {
		return
	}

	n.mu.Lock()
	now := n.now()
	if n.sending || (!n.lastSent.IsZero() && now.Sub(n.lastSent) < n.cooldown) {
		n.mu.Unlock()
		return
	}
	n.sending = true
	n.mu.Unlock()

	go n.send(clientIP, now)
}

func (n *shareLimitNotifier) send(clientIP string, sentAt time.Time) {
	delivered := false
	defer func() {
		n.mu.Lock()
		n.sending = false
		if delivered {
			n.lastSent = sentAt
		}
		n.mu.Unlock()
	}()

	body, err := json.Marshal(map[string]string{
		"state":  "warning",
		"host":   "draw.meatbags.ru",
		"unit":   "anonymous-share-rate-limit",
		"detail": "Anonymous share creation reached a protective limit (20 per client or 100 total per hour); client=" + clientIP,
	})
	if err != nil {
		return
	}

	req, err := http.NewRequest(http.MethodPost, n.url, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Authorization", "Bearer "+n.token)
	req.Header.Set("Content-Type", "application/json")
	response, err := n.client.Do(req)
	if err != nil {
		return
	}
	_ = response.Body.Close()
	delivered = response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusMultipleChoices
}

func newShareCreateLimiter() *shareCreateLimiter {
	return &shareCreateLimiter{
		now:       time.Now,
		window:    shareRateWindow,
		perClient: shareRatePerClient,
		global:    shareRateGlobal,
		clients:   make(map[string]windowCount),
	}
}

func (l *shareCreateLimiter) Allow(client string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	if l.all.StartedAt.IsZero() || now.Sub(l.all.StartedAt) >= l.window {
		l.all = windowCount{StartedAt: now}
		l.clients = make(map[string]windowCount)
	}
	if l.all.Count >= l.global {
		return false
	}

	count := l.clients[client]
	if count.StartedAt.IsZero() || now.Sub(count.StartedAt) >= l.window {
		count = windowCount{StartedAt: now}
	}
	if count.Count >= l.perClient {
		return false
	}

	count.Count++
	l.clients[client] = count
	l.all.Count++
	return true
}

func clientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		for i := len(parts) - 1; i >= 0; i-- {
			candidate := strings.TrimSpace(parts[i])
			if net.ParseIP(candidate) != nil {
				return candidate
			}
		}
	}
	if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); net.ParseIP(realIP) != nil {
		return realIP
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func HandleGet(documentStore core.DocumentStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		document, err := documentStore.FindID(r.Context(), id)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Write(document.Data.Bytes())
	}
}
