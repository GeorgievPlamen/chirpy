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

type loginInput struct {
	Password string `json:"password"`
	Email    string `json:"email"`
}

type createUserResponse struct {
	Id          string    `json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Email       string    `json:"email"`
	IsChirpyRed bool      `json:"is_chirpy_red"`
}

type loginUserResponse struct {
	Id           string    `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Email        string    `json:"email"`
	Token        string    `json:"token"`
	RefreshToken string    `json:"refresh_token"`
	IsChirpyRed  bool      `json:"is_chirpy_red"`
}

func handleLogin(w http.ResponseWriter, r *http.Request, apiCfg *apiConfig) {
	decoder := json.NewDecoder(r.Body)
	input := loginInput{}
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

	token, err := auth.MakeJWT(user.ID, apiCfg.jwtSecret)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("could not create JWT"))
		return
	}

	refreshToken, err := apiCfg.db.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{
		Token:     auth.MakeRefreshToken(),
		UserID:    user.ID,
		ExpiresAt: time.Now().UTC().Add(60 * (time.Hour * 24)),
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("could not save refresh token to db"))
		return
	}

	userRes := loginUserResponse{
		Id:           user.ID.String(),
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
		Email:        user.Email,
		Token:        token,
		RefreshToken: refreshToken.Token,
		IsChirpyRed:  user.IsChirpyRed,
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
		Id:          user.ID.String(),
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
		Email:       user.Email,
		IsChirpyRed: user.IsChirpyRed,
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

type RefreshResponse struct {
	Token string `json:"token"`
}

func handleRefresh(w http.ResponseWriter, r *http.Request, apiConfig *apiConfig) {
	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("No bearer token"))
		return
	}

	rec, err := apiConfig.db.GetUserFromRefreshToken(r.Context(), refreshToken)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("No bearer token in db"))
		return
	}

	if rec.RevokedAt.Valid || rec.ExpiresAt.UTC().Compare(time.Now().UTC()) <= 0 {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Refresh token expired or revoked"))
		return
	}

	jwt, err := auth.MakeJWT(rec.UserID, apiConfig.jwtSecret)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Something went wrong"))
	}

	res := RefreshResponse{
		Token: jwt,
	}

	resBytes, err := json.Marshal(res)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Something went wrong"))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(resBytes)
}

func handleRevoke(w http.ResponseWriter, r *http.Request, apiConfig *apiConfig) {
	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("No bearer token"))
		return
	}

	_, err = apiConfig.db.Revoke(r.Context(), refreshToken)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("No token"))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func handleUpdateUser(w http.ResponseWriter, r *http.Request, apiConfig *apiConfig) {
	jwt, err := auth.GetBearerToken(r.Header)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("no bearer token"))
	}

	userId, err := auth.ValidateJWT(jwt, apiConfig.jwtSecret)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("invalid jwt"))
	}

	var input createUserInput
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&input)
	if err != nil || input.Email == "" || input.Password == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid input"))
	}

	_, err = apiConfig.db.GetUserById(r.Context(), userId)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("could not find user"))
		return
	}

	hashedPassword, err := auth.HashPassword(input.Password)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("could not hash password"))
		return
	}

	userRes, err := apiConfig.db.UpdateUser(r.Context(), database.UpdateUserParams{
		ID:             userId,
		Email:          input.Email,
		HashedPassword: hashedPassword,
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("could not save user to"))
		return
	}

	res := createUserResponse{
		Id:        userRes.ID.String(),
		CreatedAt: userRes.CreatedAt,
		UpdatedAt: userRes.UpdatedAt,
		Email:     userRes.Email,
	}
	resBytes, err := json.Marshal(res)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("could not encode res to json"))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(resBytes)
}
