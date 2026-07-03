package main

import "net/http"

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Header().Add(headerContentType, "text/plain")
	w.Header().Add(headerContentType, "charset=utf-8")

	w.Write([]byte("OK"))
}
