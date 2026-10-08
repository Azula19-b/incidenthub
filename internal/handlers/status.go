package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/Azula19-b/incidenthub/internal/database"
	"github.com/Azula19-b/incidenthub/internal/models"
)

type statusPageData struct {
	Title    string
	Metrics  models.DashboardMetrics
	Services []models.Service
	Active   []models.Incident
	Resolved []models.Incident
}

func (a *App) registerStatusRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /status", a.statusPage)
	mux.HandleFunc("GET /api/v1/status", a.statusJSON)
	mux.HandleFunc("GET /api/v1/services", a.servicesJSON)
	mux.HandleFunc("GET /api/v1/incidents", a.incidentsJSON)
}

func (a *App) statusPage(w http.ResponseWriter, r *http.Request) {
	data, err := a.loadStatusData(r)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	a.render(w, "status.html", data)
}

func (a *App) statusJSON(w http.ResponseWriter, r *http.Request) {
	data, err := a.loadStatusData(r)
	if err != nil {
		a.writeJSONError(w, http.StatusInternalServerError, "status data unavailable")
		return
	}
	a.writeJSON(w, http.StatusOK, map[string]any{
		"overall_status": data.Metrics.OverallStatus, "services": data.Services,
		"active_incidents": data.Active, "recent_resolved_incidents": data.Resolved,
	})
}

func (a *App) servicesJSON(w http.ResponseWriter, r *http.Request) {
	services, err := a.store.ListServices(r.Context(), r.URL.Query().Get("status"))
	if err != nil {
		a.writeJSONError(w, http.StatusInternalServerError, "services unavailable")
		return
	}
	a.writeJSON(w, http.StatusOK, services)
}

func (a *App) incidentsJSON(w http.ResponseWriter, r *http.Request) {
	filter := database.IncidentFilter{Search: r.URL.Query().Get("q"), Severity: r.URL.Query().Get("severity"),
		Status: r.URL.Query().Get("status"), Sort: r.URL.Query().Get("sort")}
	incidents, err := a.store.ListIncidents(r.Context(), filter)
	if err != nil {
		a.writeJSONError(w, http.StatusInternalServerError, "incidents unavailable")
		return
	}
	a.writeJSON(w, http.StatusOK, incidents)
}

func (a *App) loadStatusData(r *http.Request) (statusPageData, error) {
	metrics, err := a.store.DashboardMetrics(r.Context())
	if err != nil {
		return statusPageData{}, err
	}
	services, err := a.store.ListServices(r.Context(), "")
	if err != nil {
		return statusPageData{}, err
	}
	active, err := a.store.ListIncidents(r.Context(), database.IncidentFilter{})
	if err != nil {
		return statusPageData{}, err
	}
	resolved := make([]models.Incident, 0, 3)
	activeOnly := make([]models.Incident, 0)
	for _, incident := range active {
		if incident.Status == models.IncidentResolved {
			if len(resolved) < 3 {
				resolved = append(resolved, incident)
			}
		} else {
			activeOnly = append(activeOnly, incident)
		}
	}
	return statusPageData{Title: "System Status", Metrics: metrics, Services: services, Active: activeOnly, Resolved: resolved}, nil
}

func (a *App) writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		return
	}
}

func (a *App) writeJSONError(w http.ResponseWriter, status int, message string) {
	a.writeJSON(w, status, map[string]string{"error": message})
}
