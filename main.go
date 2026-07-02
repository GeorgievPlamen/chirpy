package main

import (
	"net/http"
)

func main() {
	handler := http.NewServeMux()
	server := http.Server{
		Handler: handler,
		Addr:    ":8080",
	}

	handler.Handle("/app/", http.StripPrefix("/app", http.FileServer(http.Dir("."))))
	handler.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Add("Content-Type", "text/plain")
		w.Header().Add("Content-Type", "charset=utf-8")

		w.Write([]byte("OK"))
	})

	server.ListenAndServe()
}
