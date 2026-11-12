package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"machole/internal/proxy"
)

func TestDashboardServerStatus(t *testing.T) {
	p := proxy.NewServer("1.1.1.1:53")
	srv := NewHTTPServer("127.0.0.1:0", p)
	srv.Start()
	defer srv.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	w := httptest.NewRecorder()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(rw).Encode(p.Status())
	})

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", w.Code)
	}

	var st proxy.Status
	if err := json.NewDecoder(w.Body).Decode(&st); err != nil {
		t.Fatalf("Failed to decode status: %v", err)
	}

	if st.Version != proxy.Version {
		t.Errorf("Expected version %s, got %s", proxy.Version, st.Version)
	}
}

func TestDashboardUIEmbed(t *testing.T) {
	data, err := uiFS.ReadFile("ui.html")
	if err != nil {
		t.Fatalf("Failed to read ui.html from embedded FS: %v", err)
	}
	if len(data) == 0 {
		t.Fatalf("ui.html is empty")
	}
}
