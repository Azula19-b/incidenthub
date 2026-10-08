package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/Azula19-b/incidenthub/internal/database"
	"github.com/Azula19-b/incidenthub/internal/models"
	"github.com/Azula19-b/incidenthub/internal/validation"
)

var severities = []models.Severity{models.SeverityCritical, models.SeverityHigh, models.SeverityMedium, models.SeverityLow}
var incidentStatuses = []models.IncidentStatus{models.IncidentInvestigating, models.IncidentIdentified, models.IncidentMonitoring, models.IncidentResolved}

type incidentsPageData struct {
	pageData
	Incidents []models.Incident
	Notice    string
}

type incidentFormData struct {
	pageData
	Incident           models.Incident
	Services           []models.Service
	SelectedServiceIDs []int64
	Severities         []models.Severity
	Statuses           []models.IncidentStatus
	Errors             map[string]string
	IsEdit             bool
}

type incidentDetailData struct {
	pageData
	Incident models.Incident
	Notice   string
}

func (a *App) registerIncidentRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /incidents", a.listIncidents)
	mux.HandleFunc("GET /incidents/new", a.newIncident)
	mux.HandleFunc("POST /incidents", a.createIncident)
	mux.HandleFunc("GET /incidents/{id}", a.viewIncident)
	mux.HandleFunc("GET /incidents/{id}/edit", a.editIncident)
	mux.HandleFunc("POST /incidents/{id}", a.updateIncident)
	mux.HandleFunc("POST /incidents/{id}/delete", a.deleteIncident)
}

func (a *App) listIncidents(w http.ResponseWriter, r *http.Request) {
	incidents, err := a.store.ListIncidents(r.Context(), database.IncidentFilter{})
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	a.render(w, "incidents.html", incidentsPageData{
		pageData: pageData{Title: "Incidents", CurrentPage: "incidents"}, Incidents: incidents,
		Notice: r.URL.Query().Get("notice"),
	})
}

func (a *App) newIncident(w http.ResponseWriter, r *http.Request) {
	a.renderIncidentForm(w, r, models.Incident{Severity: models.SeverityMedium, Status: models.IncidentInvestigating}, nil, nil, false)
}

func (a *App) createIncident(w http.ResponseWriter, r *http.Request) {
	incident, serviceIDs, validationErrors, ok := parseIncidentForm(r)
	if !ok {
		a.renderIncidentForm(w, r, incident, serviceIDs, validationErrors, false)
		return
	}
	if err := a.store.CreateIncident(r.Context(), &incident, serviceIDs); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			validationErrors["service_ids"] = "One or more selected services no longer exist."
			a.renderIncidentForm(w, r, incident, serviceIDs, validationErrors, false)
			return
		}
		a.serverError(w, r, err)
		return
	}
	http.Redirect(w, r, "/incidents/"+strconv.FormatInt(incident.ID, 10)+"?notice=created", http.StatusSeeOther)
}

func (a *App) viewIncident(w http.ResponseWriter, r *http.Request) {
	incident, ok := a.incidentFromRequest(w, r)
	if !ok {
		return
	}
	a.render(w, "incident-detail.html", incidentDetailData{
		pageData: pageData{Title: incident.Title, CurrentPage: "incidents"}, Incident: incident,
		Notice: r.URL.Query().Get("notice"),
	})
}

func (a *App) editIncident(w http.ResponseWriter, r *http.Request) {
	incident, ok := a.incidentFromRequest(w, r)
	if !ok {
		return
	}
	serviceIDs := make([]int64, 0, len(incident.Services))
	for _, service := range incident.Services {
		serviceIDs = append(serviceIDs, service.ID)
	}
	a.renderIncidentForm(w, r, incident, serviceIDs, nil, true)
}

func (a *App) updateIncident(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		a.notFound(w, r)
		return
	}
	incident, serviceIDs, validationErrors, ok := parseIncidentForm(r)
	incident.ID = id
	if !ok {
		a.renderIncidentForm(w, r, incident, serviceIDs, validationErrors, true)
		return
	}
	if err := a.store.UpdateIncident(r.Context(), incident, serviceIDs); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			a.notFound(w, r)
		} else {
			a.serverError(w, r, err)
		}
		return
	}
	http.Redirect(w, r, "/incidents/"+strconv.FormatInt(id, 10)+"?notice=updated", http.StatusSeeOther)
}

func (a *App) deleteIncident(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		a.notFound(w, r)
		return
	}
	if err := a.store.DeleteIncident(r.Context(), id); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			a.notFound(w, r)
		} else {
			a.serverError(w, r, err)
		}
		return
	}
	http.Redirect(w, r, "/incidents?notice=deleted", http.StatusSeeOther)
}

func (a *App) renderIncidentForm(w http.ResponseWriter, r *http.Request, incident models.Incident, selected []int64, validationErrors map[string]string, edit bool) {
	services, err := a.store.ListServices(r.Context(), "")
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	title := "Report Incident"
	if edit {
		title = "Edit Incident"
	}
	a.render(w, "incident-form.html", incidentFormData{
		pageData: pageData{Title: title, CurrentPage: "incidents"}, Incident: incident,
		Services: services, SelectedServiceIDs: selected, Severities: severities,
		Statuses: incidentStatuses, Errors: validationErrors, IsEdit: edit,
	})
}

func (a *App) incidentFromRequest(w http.ResponseWriter, r *http.Request) (models.Incident, bool) {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		a.notFound(w, r)
		return models.Incident{}, false
	}
	incident, err := a.store.GetIncident(r.Context(), id)
	if errors.Is(err, database.ErrNotFound) {
		a.notFound(w, r)
		return models.Incident{}, false
	}
	if err != nil {
		a.serverError(w, r, err)
		return models.Incident{}, false
	}
	return incident, true
}

func parseIncidentForm(r *http.Request) (models.Incident, []int64, map[string]string, bool) {
	if err := r.ParseForm(); err != nil {
		return models.Incident{}, nil, map[string]string{"form": "The submitted form could not be read."}, false
	}
	serviceIDs := make([]int64, 0, len(r.Form["service_ids"]))
	seen := make(map[int64]bool)
	for _, value := range r.Form["service_ids"] {
		id, err := parseID(value)
		if err == nil && !seen[id] {
			serviceIDs = append(serviceIDs, id)
			seen[id] = true
		}
	}
	input := validation.IncidentInput{
		Title: strings.TrimSpace(r.FormValue("title")), Description: strings.TrimSpace(r.FormValue("description")),
		Reporter: strings.TrimSpace(r.FormValue("reporter")), AssignedOwner: strings.TrimSpace(r.FormValue("assigned_owner")),
		Severity: models.Severity(r.FormValue("severity")), Status: models.IncidentStatus(r.FormValue("status")), ServiceIDs: serviceIDs,
	}
	errors := validation.ValidateIncident(input)
	incident := models.Incident{Title: input.Title, Description: input.Description, Reporter: input.Reporter,
		AssignedOwner: input.AssignedOwner, Severity: input.Severity, Status: input.Status}
	return incident, serviceIDs, errors, len(errors) == 0
}
