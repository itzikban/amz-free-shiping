package httpapi

import (
	"encoding/json"
	"net/http"

	"free-ship-checker-go/internal/checker"
)

func NewRouter() http.Handler {
	mux := http.NewServeMux()
	svc := checker.New()

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

	return mux
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
