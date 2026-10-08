package handlers

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/Azula19-b/incidenthub/internal/database"
	"github.com/Azula19-b/incidenthub/internal/models"
)

type App struct {
	templates *template.Template
	store     *database.Store
}

type pageData struct {
	Title       string
	CurrentPage string
	Now         time.Time
}

var serviceStatuses = []models.ServiceStatus{
	models.StatusOperational,
	models.StatusDegraded,
	models.StatusPartial,
	models.StatusMajor,
	models.StatusMaintenance,
}

func New(databasePath string) (*App, error) {
	functions := template.FuncMap{
		"statusClass": func(value any) string {
			class := strings.ToLower(fmt.Sprint(value))
			class = strings.NewReplacer(" ", "-", "-", "").Replace(class)
			return class
		},
		"formatDate":      func(value time.Time) string { return value.Format("Jan 2, 2006") },
		"formatTimestamp": func(value time.Time) string { return value.Format("Jan 2, 2006 · 3:04 PM") },
		"containsID": func(ids []int64, id int64) bool {
			for _, value := range ids {
				if value == id {
					return true
				}
			}
			return false
		},
	}
	templates, err := template.New("pages").Funcs(functions).ParseGlob("templates/*.html")
	if err != nil {
		return nil, err
	}
	store, err := database.Open(databasePath)
	if err != nil {
		return nil, err
	}
	if err := store.Seed(context.Background()); err != nil {
		store.Close()
		return nil, err
	}
	return &App{templates: templates, store: store}, nil
}

func (a *App) Close() error { return a.store.Close() }

func (a *App) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.HandleFunc("GET /healthz", a.health)
	mux.HandleFunc("GET /", a.dashboard)
	a.registerServiceRoutes(mux)
	a.registerIncidentRoutes(mux)
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
