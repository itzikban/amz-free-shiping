package httpapi

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"free-ship-checker-go/internal/admin"
	"free-ship-checker-go/internal/altcache"
	"free-ship-checker-go/internal/checker"
	"free-ship-checker-go/internal/monitor"
	"free-ship-checker-go/internal/userpanel"
)

// NewRouter constructs and returns an http.Handler that serves the application's HTTP API.
// It registers endpoints for health checks, URL checking, monitoring, user panel (v1/me) and admin actions,
// and wires the checker, monitor, userpanel, and admin services used by those routes.
// altCache can be nil, in which case caching is disabled.
func NewRouter(altCache interface{}) http.Handler {
	mux := http.NewServeMux()
	svc := checker.New()
	svc.AltCache = altCache
	msvc := monitor.New(svc)
	usvc := userpanel.New(svc)
	asvc := admin.NewService(msvc, usvc)

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
		method := r.URL.Query().Get("method")
		if url == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "missing url query param"})
			return
		}
		res, err := svc.CheckURLWithMethod(r.Context(), url, country, zip, method)
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

	mux.HandleFunc("/v1/me/tracked-items/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowed(w, http.MethodGet)
			return
		}
		id := strings.TrimPrefix(r.URL.Path, "/v1/me/tracked-items/")
		if id == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "missing id"})
			return
		}
		item, ok := usvc.GetItem(id)
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "tracked item not found"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"item": item})
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
				writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
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

	mux.HandleFunc("/v1/me/notifications", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowed(w, http.MethodGet)
			return
		}
		unreadOnly := strings.EqualFold(r.URL.Query().Get("unread"), "true")
		writeJSON(w, http.StatusOK, map[string]any{"notifications": usvc.ListNotifications(unreadOnly, 100)})
	})

	mux.HandleFunc("/v1/me/notifications/read", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			methodNotAllowed(w, http.MethodPost)
			return
		}
		var req struct {
			ID  string `json:"id"`
			All bool   `json:"all"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
			return
		}
		if req.All {
			count := usvc.MarkAllNotificationsRead()
			writeJSON(w, http.StatusOK, map[string]any{"ok": true, "updated": count})
			return
		}
		if req.ID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "id or all=true is required"})
			return
		}
		if !usvc.MarkNotificationRead(req.ID) {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "notification not found"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	})

	mux.HandleFunc("/v1/me/notification-preferences", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			writeJSON(w, http.StatusOK, usvc.NotificationPreferences())
			return
		}
		if r.Method == http.MethodPut {
			var req userpanel.NotificationPreferences
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
				return
			}
			writeJSON(w, http.StatusOK, usvc.UpdateNotificationPreferences(req))
			return
		}
		methodNotAllowed(w, http.MethodGet, http.MethodPut)
	})

	mux.HandleFunc("/v1/admin/fetch-method", func(w http.ResponseWriter, r *http.Request) {
		if !requireAdmin(r, w) {
			return
		}
		if r.Method == http.MethodGet {
			m := svc.FetchMethod
			if m == "" {
				m = "auto"
			}
			writeJSON(w, http.StatusOK, map[string]any{"method": m})
			return
		}
		if r.Method == http.MethodPut {
			var req struct {
				Method string `json:"method"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
				return
			}
			if req.Method != "auto" && req.Method != "http" {
				writeJSON(w, http.StatusBadRequest, map[string]any{"error": "method must be 'auto' or 'http'"})
				return
			}
			svc.FetchMethod = req.Method
			writeJSON(w, http.StatusOK, map[string]any{"ok": true, "method": req.Method})
			return
		}
		methodNotAllowed(w, http.MethodGet, http.MethodPut)
	})

	mux.HandleFunc("/v1/admin/metrics", func(w http.ResponseWriter, r *http.Request) {
		if !requireAdmin(r, w) {
			return
		}
		if r.Method != http.MethodGet {
			methodNotAllowed(w, http.MethodGet)
			return
		}
		writeJSON(w, http.StatusOK, asvc.Snapshot())
	})

	mux.HandleFunc("/v1/admin/actions/replay-failed-jobs", func(w http.ResponseWriter, r *http.Request) {
		if !requireAdmin(r, w) {
			return
		}
		if r.Method != http.MethodPost {
			methodNotAllowed(w, http.MethodPost)
			return
		}
		resp := admin.ReplayFailedJobs()
		status := http.StatusOK
		if !resp.OK {
			status = http.StatusNotImplemented
		}
		writeJSON(w, status, resp)
	})

	mux.HandleFunc("/v1/admin/actions/retry-failed-notifications", func(w http.ResponseWriter, r *http.Request) {
		if !requireAdmin(r, w) {
			return
		}
		if r.Method != http.MethodPost {
			methodNotAllowed(w, http.MethodPost)
			return
		}
		processed, err := usvc.RetryFailedNotifications(r.Context(), 100)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]any{"ok": false, "action": "retry_failed_notifications", "error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "action": "retry_failed_notifications", "processed": processed})
	})

	// Alternative cache metrics endpoint
	if snapper, ok := altCache.(altcache.Snapshotter); ok {
		mux.HandleFunc("/v1/admin/altcache/metrics", func(w http.ResponseWriter, r *http.Request) {
			if !requireAdmin(r, w) {
				return
			}
			if r.Method != http.MethodGet {
				methodNotAllowed(w, http.MethodGet)
				return
			}
			writeJSON(w, http.StatusOK, snapper.Snapshot())
		})
	}

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

func requireAdmin(r *http.Request, w http.ResponseWriter) bool {
	adminToken := strings.TrimSpace(os.Getenv("ADMIN_API_TOKEN"))
	if adminToken == "" {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "admin access is disabled"})
		return false
	}
	provided := strings.TrimSpace(r.Header.Get("X-Admin-Token"))
	if provided == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "missing admin credentials"})
		return false
	}
	if subtle.ConstantTimeCompare([]byte(provided), []byte(adminToken)) != 1 {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "forbidden"})
		return false
	}
	return true
}
