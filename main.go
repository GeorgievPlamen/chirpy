package main

import (
	"database/sql"
	"os"

	"github.com/georgievplamen/chirpy/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"log"
	"net/http"
)

const (
	headerContentType     = "Content-Type"
	failedToWriteResponse = "failed to write response: %v"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Printf("Could not load env variables: %v", err)
	}

	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Printf("Could not connect to db: %v", err)
	}

	apiCfg := &apiConfig{
		db:        database.New(db),
		platform:  os.Getenv("PLATFORM"),
		jwtSecret: os.Getenv("PLATJWT_SECRETFORM"),
		polkaKey:  os.Getenv("POLKA_KEY"),
	}

	handler := http.NewServeMux()
	server := &http.Server{
		Handler: handler,
		Addr:    ":8080",
	}
	handler.Handle("/app/", http.StripPrefix("/app", apiCfg.middlewareMetricsInc(http.FileServer(http.Dir(".")))))
	handler.HandleFunc("GET /api/healthz", handleHealth)
	handler.HandleFunc("GET /admin/metrics", apiCfg.handleMetrics)
	handler.HandleFunc("POST /admin/reset", apiCfg.handleReset)
	handler.HandleFunc("POST /api/users", apiCfg.handle(handleCreateUser))
	handler.HandleFunc("POST /api/chirps", apiCfg.handle(handleCreateChirp))
	handler.HandleFunc("GET /api/chirps", apiCfg.handle(handleGetChirps))
	handler.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.handle(handleGetChirpById))
	handler.HandleFunc("POST /api/login", apiCfg.handle(handleLogin))
	handler.HandleFunc("POST /api/refresh", apiCfg.handle(handleRefresh))
	handler.HandleFunc("POST /api/revoke", apiCfg.handle(handleRevoke))
	handler.HandleFunc("PUT /api/users", apiCfg.handle(handleUpdateUser))
	handler.HandleFunc("DELETE /api/chirps/{chirpID}", apiCfg.handle(handleDeleteById))
	handler.HandleFunc("POST /api/polka/webhooks", apiCfg.handle(handlerPolkaWebhook))

	server.ListenAndServe()
}
