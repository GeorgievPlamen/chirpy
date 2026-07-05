package main

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
)

type webhookRequest struct {
	Event string `json:"event"`
}
type polkaWebhookRequest struct {
	webhookRequest
	Data struct {
		UserId string `json:"user_id"`
	} `json:"data"`
}

func handlerPolkaWebhook(w http.ResponseWriter, r *http.Request, apiCfg *apiConfig) {
	input := polkaWebhookRequest{}
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil || input.Data.UserId == "" || input.Event == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	userId, err := uuid.Parse(input.Data.UserId)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	user, err := apiCfg.db.GetUserById(r.Context(), userId)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	user, err = apiCfg.db.UpdateUserToChirpyRed(r.Context(), user.ID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
