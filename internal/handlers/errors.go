package handlers

import (
	"log"
	"net/http"
)

func (a *App) notFound(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "The requested resource was not found.", http.StatusNotFound)
}

func (a *App) serverError(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("request %s %s failed: %v", r.Method, r.URL.Path, err)
	http.Error(w, "An unexpected error occurred.", http.StatusInternalServerError)
}
