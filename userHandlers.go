package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/georgievplamen/chirpy/internal/auth"
	"github.com/georgievplamen/chirpy/internal/database"
)

type createUserInput struct {
	Password string `json:"password"`
	Email    string `json:"email"`
}

type createUserResponse struct {
	Id        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

func handleLogin(w http.ResponseWriter, r *http.Request, apiCfg *apiConfig) {
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

	if input.Password == "" {
		log.Printf("You need to provide a password")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	user, err := apiCfg.db.GetUserByEmail(r.Context(), input.Email)
	if err != nil {
		log.Printf("\n Could not get user info: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	match, err := auth.CheckPasswordHash(input.Password, user.HashedPassword)
	if err != nil || !match {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Incorrect email or password"))
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

	w.WriteHeader(http.StatusOK)
	w.Write(userJson)
}

func handleCreateUser(w http.ResponseWriter, r *http.Request, apiCfg *apiConfig) {
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

	if input.Password == "" {
		log.Printf("You need to provide a password")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	hash, err := auth.HashPassword(input.Password)
	if err != nil {
		log.Printf("Password could not be hashed")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	user, err := apiCfg.db.CreateUser(r.Context(), database.CreateUserParams{
		Email:          input.Email,
		HashedPassword: hash,
	})
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
}
