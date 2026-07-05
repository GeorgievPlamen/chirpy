package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/georgievplamen/chirpy/internal/auth"
	"github.com/georgievplamen/chirpy/internal/database"
	"github.com/google/uuid"
)

type createChirpRequest struct {
	Body string `json:"body"`
}

type errorRes struct {
	Error string `json:"error"`
}

type chirpResponse struct {
	Id        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserId    string    `json:"user_id"`
}

func handleDeleteById(w http.ResponseWriter, r *http.Request, apiCfg *apiConfig) {
	jwt, err := auth.GetBearerToken(r.Header)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		errRes := errorRes{
			Error: "Invalid JWT",
		}

		responseBytes, err := json.Marshal(errRes)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if _, err := w.Write(responseBytes); err != nil {
			log.Printf(failedToWriteResponse, err)
		}
	}

	userId, err := auth.ValidateJWT(jwt, apiCfg.jwtSecret)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		errRes := errorRes{
			Error: "Invalid JWT",
		}

		responseBytes, err := json.Marshal(errRes)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if _, err := w.Write(responseBytes); err != nil {
			log.Printf(failedToWriteResponse, err)
		}
	}

	chirpIdRaw := r.PathValue("chirpID")
	if chirpIdRaw == "" {
		respondWithError(w, "You need to provied a chirp ID")
		return
	}
	chirpId, err := uuid.Parse(chirpIdRaw)
	if err != nil {
		respondWithError(w, "You need to provied a valid chirp ID")
		return
	}

	chirp, err := apiCfg.db.GetChirpById(r.Context(), chirpId)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			w.WriteHeader(http.StatusNotFound)
		} else {
			respondWithError(w, err.Error())
		}
		return
	}

	if chirp.UserID != userId {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	err = apiCfg.db.DeleteChirpById(r.Context(), chirpId)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			w.WriteHeader(http.StatusNotFound)
		} else {
			respondWithError(w, err.Error())
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func handleGetChirpById(w http.ResponseWriter, r *http.Request, apiCfg *apiConfig) {
	chirpIdRaw := r.PathValue("chirpID")
	id, err := uuid.Parse(chirpIdRaw)
	if err != nil {
		respondWithError(w, err.Error())
		return
	}

	chirp, err := apiCfg.db.GetChirpById(r.Context(), id)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			w.WriteHeader(http.StatusNotFound)
		} else {
			respondWithError(w, err.Error())
		}
		return
	}

	res := chirpResponse{
		Id:        chirp.ID.String(),
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body:      chirp.Body,
		UserId:    chirp.UserID.String(),
	}

	resBytes, err := json.Marshal(res)
	if err != nil {
		respondWithError(w, err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(resBytes)
}

func handleGetChirps(w http.ResponseWriter, r *http.Request, apiCfg *apiConfig) {
	queryParams := r.URL.Query()
	authorIdParams := queryParams["author_id"]

	if len(authorIdParams) > 0 && authorIdParams[0] != "" {
		authorId, err := uuid.Parse(authorIdParams[0])
		if err != nil {
			respondWithError(w, err.Error())
			return
		}

		chirps, err := apiCfg.db.GetChirpsByAuthorId(r.Context(), authorId)
		if err != nil {
			respondWithError(w, err.Error())
			return
		}

		res := make([]chirpResponse, 0)
		for _, v := range chirps {
			res = append(res, chirpResponse{
				Id:        v.ID.String(),
				CreatedAt: v.CreatedAt,
				UpdatedAt: v.UpdatedAt,
				Body:      v.Body,
				UserId:    v.UserID.String(),
			})
		}

		resBytes, err := json.Marshal(res)
		if err != nil {
			respondWithError(w, err.Error())
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(resBytes)
		return
	}

	chirps, err := apiCfg.db.GetChirps(r.Context())
	if err != nil {
		respondWithError(w, err.Error())
		return
	}

	res := make([]chirpResponse, 0)
	for _, v := range chirps {
		res = append(res, chirpResponse{
			Id:        v.ID.String(),
			CreatedAt: v.CreatedAt,
			UpdatedAt: v.UpdatedAt,
			Body:      v.Body,
			UserId:    v.UserID.String(),
		})
	}

	sortParam := queryParams.Get("sort")

	if sortParam == "asc" {
		sort.Slice(res, func(i, j int) bool {
			return i < j
		})
	}

	if sortParam == "desc" {
		sort.Slice(res, func(i, j int) bool {
			return i > j
		})
	}

	resBytes, err := json.Marshal(res)
	if err != nil {
		respondWithError(w, err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(resBytes)
}

func handleCreateChirp(w http.ResponseWriter, r *http.Request, apiCfg *apiConfig) {
	jwt, err := auth.GetBearerToken(r.Header)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		errRes := errorRes{
			Error: "Invalid JWT",
		}

		responseBytes, err := json.Marshal(errRes)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if _, err := w.Write(responseBytes); err != nil {
			log.Printf(failedToWriteResponse, err)
		}
	}

	userId, err := auth.ValidateJWT(jwt, apiCfg.jwtSecret)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		errRes := errorRes{
			Error: "Invalid JWT",
		}

		responseBytes, err := json.Marshal(errRes)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if _, err := w.Write(responseBytes); err != nil {
			log.Printf(failedToWriteResponse, err)
		}
	}

	chirp := createChirpRequest{}
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&chirp)
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
		respondWithError(w, "Chirp is too long")
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

	createdChirp, err := apiCfg.db.CreateChirp(r.Context(), database.CreateChirpParams{
		Body:   cleanedBody.String(),
		UserID: userId,
	})
	if err != nil {
		respondWithError(w, err.Error())
		return
	}

	res := chirpResponse{
		Id:        createdChirp.ID.String(),
		CreatedAt: createdChirp.CreatedAt,
		UpdatedAt: createdChirp.UpdatedAt,
		Body:      createdChirp.Body,
		UserId:    createdChirp.UserID.String(),
	}

	resBytes, err := json.Marshal(res)
	if err != nil {
		respondWithError(w, err.Error())
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write(resBytes)
}

func respondWithError(w http.ResponseWriter, errText string) {
	w.WriteHeader(http.StatusBadRequest)
	errRes := errorRes{
		Error: errText,
	}

	responseBytes, err := json.Marshal(errRes)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if _, err := w.Write(responseBytes); err != nil {
		log.Printf(failedToWriteResponse, err)
	}
}
