package main

import (
	"fmt"
	"log"
	"net/http"
	"sync/atomic"

	"github.com/georgievplamen/chirpy/internal/database"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	db             *database.Queries
	platform       string
	jwtSecret      string
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) handle(handler func(w http.ResponseWriter, r *http.Request, apiCfg *apiConfig)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		handler(w, r, cfg)
	}
}

func (cfg *apiConfig) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(headerContentType, "text/html; charset=utf-8")

	html := fmt.Sprintf(`
<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, cfg.fileserverHits.Load())

	if _, err := w.Write([]byte(html)); err != nil {
		log.Printf("failed to write metrics response: %v", err)
	}
}

func (cfg *apiConfig) handleReset(w http.ResponseWriter, r *http.Request) {

	if cfg.platform != "dev" {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	cfg.fileserverHits = atomic.Int32{}
	err := cfg.db.ResetUsers(r.Context())
	if err != nil {
		fmt.Printf("\n Could not reset users: %v", err)
	}
}
