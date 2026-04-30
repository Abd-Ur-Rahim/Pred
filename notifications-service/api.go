package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"gorm.io/gorm"
)

func startHTTPServer(gdb *gorm.DB) {
	mux := http.NewServeMux()
	mux.HandleFunc("/notifications", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		tenantID := r.URL.Query().Get("tenant_id")
		if tenantID == "" {
			http.Error(w, "tenant_id is required", http.StatusBadRequest)
			return
		}

		limitStr := r.URL.Query().Get("limit")

		limit := 10
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
		if limit <= 0 {
			limit = 10
		}
		if limit > 100 {
			limit = 100
		}

		var notifs []map[string]interface{}

		err := gdb.Raw(`
			SELECT id, tenant_id, type, payload, created_at
			FROM notifications
			WHERE tenant_id = ?
			ORDER BY created_at DESC
			LIMIT ?
		`, tenantID, limit).Scan(&notifs).Error

		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(notifs)
	})

	server := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("http server failed: %v", err)
		}
	}()
}