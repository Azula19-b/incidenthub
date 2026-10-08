package handlers

import (
	"log"
	"net/http"
)

func (a *App) notFound(w http.ResponseWriter, r *http.Request) {
	a.renderStatus(w, "404.html", pageData{Title: "Page Not Found"}, http.StatusNotFound)
}

func (a *App) serverError(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("request %s %s failed: %v", r.Method, r.URL.Path, err)
	a.renderStatus(w, "500.html", pageData{Title: "Something Went Wrong"}, http.StatusInternalServerError)
}
