package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/y16ra/zipcode-da-monorepo/internal/client"
	"github.com/y16ra/zipcode-da-monorepo/internal/config"
	"github.com/y16ra/zipcode-da-monorepo/internal/handler"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	jp := client.NewJapanPost(cfg)
	searchH := &handler.SearchZipcodeHandler{JP: jp, DefaultECUID: cfg.JapanPostECUID}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", handler.Healthz)
	mux.Handle("/api/v1/search/zipcode", corsMiddleware(http.HandlerFunc(searchH.ServeHTTP)))

	addr := cfg.HTTPAddr
	log.Printf("listening on %s", addr)
	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server: %v", err)
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	allow := os.Getenv("CORS_ALLOW_ORIGIN")
	if allow == "" {
		allow = "*"
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", allow)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
