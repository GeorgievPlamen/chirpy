package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

type createUserInput struct {
	Email string `json:"email"`
}

type createUserResponse struct {
	Id        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
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
}
