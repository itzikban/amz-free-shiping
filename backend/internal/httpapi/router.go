package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"free-ship-checker-go/internal/checker"
	"free-ship-checker-go/internal/monitor"
	"free-ship-checker-go/internal/userpanel"
)

func NewRouter() http.Handler {
	mux := http.NewServeMux()
	svc := checker.New()
	msvc := monitor.New(svc)
	usvc := userpanel.New(svc)

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowed(w, http.MethodGet)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	})

	mux.HandleFunc("/check", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowed(w, http.MethodGet)
			return
		}
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
		if r.Method != http.MethodGet {
			methodNotAllowed(w, http.MethodGet)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"monitors": msvc.List()})
	})

	mux.HandleFunc("/monitor/notifications", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			msvc.ClearNotifications()
			writeJSON(w, http.StatusOK, map[string]any{"ok": true})
			return
		}
		if r.Method != http.MethodGet {
			methodNotAllowed(w, http.MethodGet, http.MethodDelete)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"notifications": msvc.Notifications()})
	})

	mux.HandleFunc("/monitor/clear", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			methodNotAllowed(w, http.MethodDelete)
			return
		}
		msvc.ClearMonitors()
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	})

	mux.HandleFunc("/v1/me", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowed(w, http.MethodGet)
			return
		}
		writeJSON(w, http.StatusOK, usvc.Me())
	})

	mux.HandleFunc("/v1/me/tracked-items", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			writeJSON(w, http.StatusOK, map[string]any{"items": usvc.ListItems()})
			return
		}
		if r.Method == http.MethodPost {
			var req userpanel.AddTrackedItemReq
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
				return
			}
			if req.URL == "" {
				writeJSON(w, http.StatusBadRequest, map[string]any{"error": "url is required"})
				return
			}
			if req.Country == "" {
				req.Country = "US"
			}
			item, err := usvc.AddTrackedItem(r.Context(), req)
			if err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, item)
			return
		}
		methodNotAllowed(w, http.MethodGet, http.MethodPost)
	})

	mux.HandleFunc("/v1/me/alerts", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowed(w, http.MethodGet)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"alerts": usvc.ListAlerts()})
	})

	return mux
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func methodNotAllowed(w http.ResponseWriter, allowed ...string) {
	if len(allowed) > 0 {
		w.Header().Set("Allow", strings.Join(allowed, ", "))
	}
	writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
}
