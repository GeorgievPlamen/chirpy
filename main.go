package main

import (
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

const (
	headerContentType = "Content-Type"
)

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
	cfg.fileserverHits = atomic.Int32{}
}

func main() {
	handler := http.NewServeMux()
	server := &http.Server{
		Handler: handler,
		Addr:    ":8080",
	}

	apiCfg := &apiConfig{}

	handler.Handle("/app/", http.StripPrefix("/app", apiCfg.middlewareMetricsInc(http.FileServer(http.Dir(".")))))
	handler.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Add(headerContentType, "text/plain")
		w.Header().Add(headerContentType, "charset=utf-8")

		w.Write([]byte("OK"))
	})
	handler.HandleFunc("GET /admin/metrics", apiCfg.handleMetrics)
	handler.HandleFunc("POST /admin/reset", apiCfg.handleReset)

	server.ListenAndServe()
}
