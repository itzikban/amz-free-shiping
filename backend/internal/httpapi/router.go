package httpapi

import (
	"encoding/json"
	"net/http"

	"free-ship-checker-go/internal/checker"
	"free-ship-checker-go/internal/monitor"
)

func NewRouter() http.Handler {
	mux := http.NewServeMux()
	svc := checker.New()
	msvc := monitor.New(svc)

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	})

	mux.HandleFunc("/check", func(w http.ResponseWriter, r *http.Request) {
		url := r.URL.Query().Get("url")
		country := r.URL.Query().Get("country")
		zip := r.URL.Query().Get("zip")
		if url == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "missing url query param"})
			return
		}
		res, err := svc.CheckURL(r.Context(), url, country, zip)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, res)
	})

	mux.HandleFunc("/monitor/start", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
			return
		}
		var req monitor.StartReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
			return
		}
		m, err := msvc.Start(r.Context(), req)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, m)
	})

	mux.HandleFunc("/monitor/stop", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
			return
		}
		id := r.URL.Query().Get("id")
		if id == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "missing id"})
			return
		}
		msvc.Stop(id)
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	})

	mux.HandleFunc("/monitor/list", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"monitors": msvc.List()})
	})

	mux.HandleFunc("/monitor/notifications", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"notifications": msvc.Notifications()})
	})

	return mux
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
