package server

import (
	"embed"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"machole/internal/proxy"
)

//go:embed ui.html
var uiFS embed.FS

// HTTPServer represents the embedded dashboard and API server.
type HTTPServer struct {
	addr   string
	proxy  *proxy.Server
	server *http.Server
}

// NewHTTPServer constructs an HTTP dashboard server for MacHole.
func NewHTTPServer(addr string, p *proxy.Server) *HTTPServer {
	return &HTTPServer{
		addr:  addr,
		proxy: p,
	}
}

func originFilter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			if !strings.HasPrefix(origin, "http://127.0.0.1:") &&
				!strings.HasPrefix(origin, "http://localhost:") &&
				!strings.HasPrefix(origin, "https://127.0.0.1:") &&
				!strings.HasPrefix(origin, "https://localhost:") {
				http.Error(w, "Forbidden origin", http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// Start begins serving the dashboard and API asynchronously.
func (s *HTTPServer) Start() {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		data, err := uiFS.ReadFile("ui.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(data)
	})

	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(s.proxy.Status())
	})

	mux.HandleFunc("/api/start", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			ListenAddr string `json:"listen_addr"`
			Interface  string `json:"interface"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if req.ListenAddr == "" {
			req.ListenAddr = "127.0.0.1:1080"
		}
		if req.Interface == "" {
			req.Interface = "en0"
		}
		err := s.proxy.Start(req.ListenAddr, req.Interface)
		w.Header().Set("Content-Type", "application/json")
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
		}
		_ = json.NewEncoder(w).Encode(s.proxy.Status())
	})

	mux.HandleFunc("/api/stop", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		s.proxy.Stop()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(s.proxy.Status())
	})

	mux.HandleFunc("/api/check-ip", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		res, err := s.proxy.CheckExternalIP()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		_, _ = w.Write([]byte(res))
	})

	s.server = &http.Server{
		Addr:    s.addr,
		Handler: originFilter(mux),
	}

	go func() {
		if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("UI server error: %v", err)
		}
	}()
}

// Close gracefully stops the HTTP server.
func (s *HTTPServer) Close() error {
	if s.server != nil {
		return s.server.Close()
	}
	return nil
}
