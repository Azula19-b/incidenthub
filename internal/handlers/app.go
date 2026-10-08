package handlers

import (
	"bytes"
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

type dashboardPageData struct {
	pageData
	Metrics           models.DashboardMetrics
	RecentIncidents   []models.Incident
	Services          []models.Service
	SeverityBreakdown []models.SeverityMetric
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
		"formatHours": func(value float64) string {
			if value < 1 {
				return fmt.Sprintf("%.0f min", value*60)
			}
			return fmt.Sprintf("%.1f hrs", value)
		},
		"percent": func(value, total int) float64 {
			if total == 0 {
				return 0
			}
			return float64(value) * 100 / float64(total)
		},
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
	if err := store.RecalculateAllServiceStatuses(context.Background()); err != nil {
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
	mux.HandleFunc("GET /analytics", a.analytics)
	a.registerServiceRoutes(mux)
	a.registerIncidentRoutes(mux)
	a.registerStatusRoutes(mux)
	return mux
}

func (a *App) dashboard(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		a.notFound(w, r)
		return
	}
	metrics, err := a.store.DashboardMetrics(r.Context())
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	incidents, err := a.store.ListIncidents(r.Context(), database.IncidentFilter{})
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	if len(incidents) > 5 {
		incidents = incidents[:5]
	}
	services, err := a.store.ListServices(r.Context(), "")
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	analytics, err := a.store.Analytics(r.Context())
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	a.render(w, "dashboard.html", dashboardPageData{
		pageData: pageData{Title: "Operations Dashboard", CurrentPage: "dashboard", Now: time.Now()},
		Metrics:  metrics, RecentIncidents: incidents, Services: services, SeverityBreakdown: analytics.SeverityDistribution,
	})
}

func (a *App) analytics(w http.ResponseWriter, r *http.Request) {
	analytics, err := a.store.Analytics(r.Context())
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	a.render(w, "analytics.html", struct {
		pageData
		Analytics models.Analytics
	}{pageData: pageData{Title: "Incident Analytics", CurrentPage: "analytics"}, Analytics: analytics})
}

func (a *App) health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func (a *App) render(w http.ResponseWriter, name string, data any) {
	a.renderStatus(w, name, data, http.StatusOK)
}

func (a *App) renderStatus(w http.ResponseWriter, name string, data any, status int) {
	var output bytes.Buffer
	if err := a.templates.ExecuteTemplate(&output, name, data); err != nil {
		http.Error(w, "Unable to render page", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = output.WriteTo(w)
}
