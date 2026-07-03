package main

import (
	"database/sql"
	"os"
	"time"

	"github.com/georgievplamen/chirpy/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	db             *database.Queries
	platform       string
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
		db:       database.New(db),
		platform: os.Getenv("PLATFORM"),
	}

	handler := http.NewServeMux()
	server := &http.Server{
		Handler: handler,
		Addr:    ":8080",
	}
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

	handler.HandleFunc("POST /api/users", func(w http.ResponseWriter, r *http.Request) {
		decoder := json.NewDecoder(r.Body)
		input := createUserInput{}
		err := decoder.Decode(&input)
		if err != nil {
			log.Printf("\n Could not decode request input: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if input.Email == "" {
			log.Printf("You need to provide an email address")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		user, err := apiCfg.db.CreateUser(r.Context(), input.Email)
		if err != nil {
			log.Printf("Could not create user: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		userRes := createUserResponse{
			Id:        user.ID.String(),
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
			Email:     user.Email,
		}

		userJson, err := json.Marshal(userRes)
		if err != nil {
			log.Printf("Could not encode user to JSON: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		w.Write(userJson)
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

type createUserInput struct {
	Email string `json:"email"`
}

type createUserResponse struct {
	Id        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}
