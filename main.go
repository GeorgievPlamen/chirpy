package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
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
	headerContentType     = "Content-Type"
	failedToWriteResponse = "failed to write response: %v"
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
	handler.HandleFunc("POST /api/validate_chirp", func(w http.ResponseWriter, r *http.Request) {
		chirp := chirp{}
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&chirp)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)

			errRes := errorRes{
				Error: "Something went wrong",
			}
			responseBytes, err := json.Marshal(errRes)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			if _, err := w.Write(responseBytes); err != nil {
				log.Printf(failedToWriteResponse, err)
			}
			return
		}

		if len(chirp.Body) > 140 {
			w.WriteHeader(http.StatusBadRequest)
			errRes := errorRes{
				Error: "Chirp is too long",
			}

			responseBytes, err := json.Marshal(errRes)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			if _, err := w.Write(responseBytes); err != nil {
				log.Printf(failedToWriteResponse, err)
			}
			return
		}

		words := strings.Fields(chirp.Body)

		badWordsSet := map[string]struct{}{
			"kerfuffle": {},
			"sharbert":  {},
			"fornax":    {},
		}

		cleanedBody := strings.Builder{}
		for i, word := range words {
			if _, ok := badWordsSet[strings.ToLower(word)]; ok {
				cleanedBody.WriteString("****")
			} else {
				cleanedBody.WriteString(word)
			}

			if i < len(words)-1 {
				cleanedBody.WriteRune(' ')
			}
		}

		responseBytes, err := json.Marshal(validRes{
			CleanedBody: cleanedBody.String(),
		})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Println("Could not encode valid response")
		}

		if _, err := w.Write(responseBytes); err != nil {
			log.Printf(failedToWriteResponse, err)
		}
	})

	server.ListenAndServe()
}

type chirp struct {
	Body string `json:"body"`
}

type validRes struct {
	CleanedBody string `json:"cleaned_body"`
}

type errorRes struct {
	Error string `json:"error"`
}
