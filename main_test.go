package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"excalidraw-complete/stores/memory"
)

func TestOwnerAPICanvasRoundTrip(t *testing.T) {
	t.Setenv("DRAW_MEATBAGS_OWNER_API_TOKEN", testOwnerToken)
	t.Setenv("DRAW_MEATBAGS_OWNER_API_USER_ID", "github:5765513")

	router := setupRouter(memory.NewStore())
	canvas := `{"type":"excalidraw","version":2,"elements":[],"appState":{"name":"Owner API test"},"files":{}}`

	put := httptest.NewRequest(http.MethodPut, "/api/v2/kv/owner-api-test/", strings.NewReader(canvas))
	put.Header.Set("Authorization", "Bearer "+testOwnerToken)
	putResponse := httptest.NewRecorder()
	router.ServeHTTP(putResponse, put)
	if putResponse.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, want %d; body=%s", putResponse.Code, http.StatusOK, putResponse.Body.String())
	}

	get := httptest.NewRequest(http.MethodGet, "/api/v2/kv/owner-api-test/", nil)
	get.Header.Set("Authorization", "Bearer "+testOwnerToken)
	getResponse := httptest.NewRecorder()
	router.ServeHTTP(getResponse, get)
	if getResponse.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want %d; body=%s", getResponse.Code, http.StatusOK, getResponse.Body.String())
	}
	if !bytes.Equal(bytes.TrimSpace(getResponse.Body.Bytes()), []byte(canvas)) {
		t.Fatalf("GET body differs from saved canvas: %s", getResponse.Body.String())
	}

	list := httptest.NewRequest(http.MethodGet, "/api/v2/kv/", nil)
	list.Header.Set("Authorization", "Bearer "+testOwnerToken)
	listResponse := httptest.NewRecorder()
	router.ServeHTTP(listResponse, list)
	if listResponse.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d; body=%s", listResponse.Code, http.StatusOK, listResponse.Body.String())
	}
	var canvases []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(listResponse.Body.Bytes(), &canvases); err != nil {
		t.Fatalf("decode canvas list: %v", err)
	}
	if len(canvases) != 1 || canvases[0].ID != "owner-api-test" || canvases[0].Name != "Owner API test" {
		t.Fatalf("unexpected canvas list: %#v", canvases)
	}

	deleteRequest := httptest.NewRequest(http.MethodDelete, "/api/v2/kv/owner-api-test/", nil)
	deleteRequest.Header.Set("Authorization", "Bearer "+testOwnerToken)
	deleteResponse := httptest.NewRecorder()
	router.ServeHTTP(deleteResponse, deleteRequest)
	if deleteResponse.Code != http.StatusForbidden {
		t.Fatalf("DELETE status = %d, want %d", deleteResponse.Code, http.StatusForbidden)
	}
}

func TestAnonymousShareDocumentCreationIsRejected(t *testing.T) {
	router := setupRouter(memory.NewStore())
	req := httptest.NewRequest(http.MethodPost, "/api/v2/post/", strings.NewReader("encrypted-payload"))
	response := httptest.NewRecorder()

	router.ServeHTTP(response, req)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("POST status = %d, want %d; body=%s", response.Code, http.StatusUnauthorized, response.Body.String())
	}
}

func TestOwnerAPICanCreateShareDocument(t *testing.T) {
	t.Setenv("DRAW_MEATBAGS_OWNER_API_TOKEN", testOwnerToken)
	t.Setenv("DRAW_MEATBAGS_OWNER_API_USER_ID", "github:5765513")

	router := setupRouter(memory.NewStore())
	req := httptest.NewRequest(http.MethodPost, "/api/v2/post/", strings.NewReader("encrypted-payload"))
	req.Header.Set("Authorization", "Bearer "+testOwnerToken)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, req)

	if response.Code != http.StatusOK {
		t.Fatalf("POST status = %d, want %d; body=%s", response.Code, http.StatusOK, response.Body.String())
	}
}

const testOwnerToken = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
