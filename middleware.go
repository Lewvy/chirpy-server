package main

import (
	"net/http"
	"os"

	"github.com/Lewvy/chirpy/api"
)

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) middlewareCheckPlatform(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		platform := os.Getenv("PLATFORM")
		if platform != "dev" {
			api.RespondWithError(w, "Forbidden url", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
