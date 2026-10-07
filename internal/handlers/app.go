package handlers

import (
	"html/template"
	"net/http"
	"time"
)

type App struct {
	templates *template.Template
}

type pageData struct {
	Title       string
	CurrentPage string
	Now         time.Time
}

func New() (*App, error) {
	templates, err := template.ParseGlob("templates/*.html")
	if err != nil {
		return nil, err
	}
	return &App{templates: templates}, nil
}

func (a *App) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.HandleFunc("GET /healthz", a.health)
	mux.HandleFunc("GET /", a.dashboard)
	return mux
}

func (a *App) dashboard(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	a.render(w, "dashboard.html", pageData{
		Title:       "Operations Dashboard",
		CurrentPage: "dashboard",
		Now:         time.Now(),
	})
}

func (a *App) health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func (a *App) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := a.templates.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, "Unable to render page", http.StatusInternalServerError)
	}
}
