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

type servicesPageData struct {
	pageData
	Services       []models.Service
	SelectedStatus string
	Statuses       []models.ServiceStatus
	Notice         string
}

type serviceFormData struct {
	pageData
	Service  models.Service
	Errors   map[string]string
	Statuses []models.ServiceStatus
	IsEdit   bool
}

type serviceDetailData struct {
	pageData
	Service models.Service
	Error   string
}

func (a *App) registerServiceRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /services", a.listServices)
	mux.HandleFunc("GET /services/new", a.newService)
	mux.HandleFunc("POST /services", a.createService)
	mux.HandleFunc("GET /services/{id}", a.viewService)
	mux.HandleFunc("GET /services/{id}/edit", a.editService)
	mux.HandleFunc("POST /services/{id}", a.updateService)
	mux.HandleFunc("POST /services/{id}/delete", a.deleteService)
}

func (a *App) listServices(w http.ResponseWriter, r *http.Request) {
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	services, err := a.store.ListServices(r.Context(), status)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	a.render(w, "services.html", servicesPageData{
		pageData:       pageData{Title: "Services", CurrentPage: "services"},
		Services:       services,
		SelectedStatus: status,
		Statuses:       serviceStatuses,
		Notice:         r.URL.Query().Get("notice"),
	})
}

func (a *App) newService(w http.ResponseWriter, _ *http.Request) {
	a.render(w, "service-form.html", serviceFormData{
		pageData: pageData{Title: "Add Service", CurrentPage: "services"},
		Service:  models.Service{Status: models.StatusOperational},
		Statuses: serviceStatuses,
	})
}

func (a *App) createService(w http.ResponseWriter, r *http.Request) {
	service, validationErrors, ok := parseServiceForm(r)
	if !ok {
		a.render(w, "service-form.html", serviceFormData{
			pageData: pageData{Title: "Add Service", CurrentPage: "services"},
			Service:  service, Errors: validationErrors, Statuses: serviceStatuses,
		})
		return
	}
	if err := a.store.CreateService(r.Context(), &service); err != nil {
		if errors.Is(err, database.ErrConflict) {
			validationErrors["name"] = "A service with this name already exists."
			a.render(w, "service-form.html", serviceFormData{
				pageData: pageData{Title: "Add Service", CurrentPage: "services"},
				Service:  service, Errors: validationErrors, Statuses: serviceStatuses,
			})
			return
		}
		a.serverError(w, r, err)
		return
	}
	http.Redirect(w, r, "/services/"+strconv.FormatInt(service.ID, 10)+"?notice=created", http.StatusSeeOther)
}

func (a *App) viewService(w http.ResponseWriter, r *http.Request) {
	service, ok := a.serviceFromRequest(w, r)
	if !ok {
		return
	}
	a.render(w, "service-detail.html", serviceDetailData{
		pageData: pageData{Title: service.Name, CurrentPage: "services"},
		Service:  service,
		Error:    queryMessage(r.URL.Query().Get("error")),
	})
}

func (a *App) editService(w http.ResponseWriter, r *http.Request) {
	service, ok := a.serviceFromRequest(w, r)
	if !ok {
		return
	}
	a.render(w, "service-form.html", serviceFormData{
		pageData: pageData{Title: "Edit Service", CurrentPage: "services"},
		Service:  service, Statuses: serviceStatuses, IsEdit: true,
	})
}

func (a *App) updateService(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		a.notFound(w, r)
		return
	}
	service, validationErrors, ok := parseServiceForm(r)
	service.ID = id
	if !ok {
		a.render(w, "service-form.html", serviceFormData{
			pageData: pageData{Title: "Edit Service", CurrentPage: "services"},
			Service:  service, Errors: validationErrors, Statuses: serviceStatuses, IsEdit: true,
		})
		return
	}
	if err := a.store.UpdateService(r.Context(), service); err != nil {
		switch {
		case errors.Is(err, database.ErrNotFound):
			a.notFound(w, r)
		case errors.Is(err, database.ErrConflict):
			validationErrors["name"] = "A service with this name already exists."
			a.render(w, "service-form.html", serviceFormData{
				pageData: pageData{Title: "Edit Service", CurrentPage: "services"},
				Service:  service, Errors: validationErrors, Statuses: serviceStatuses, IsEdit: true,
			})
		default:
			a.serverError(w, r, err)
		}
		return
	}
	http.Redirect(w, r, "/services/"+strconv.FormatInt(id, 10)+"?notice=updated", http.StatusSeeOther)
}

func (a *App) deleteService(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		a.notFound(w, r)
		return
	}
	if err := a.store.DeleteService(r.Context(), id); err != nil {
		switch {
		case errors.Is(err, database.ErrNotFound):
			a.notFound(w, r)
		case errors.Is(err, database.ErrConflict):
			http.Redirect(w, r, "/services/"+strconv.FormatInt(id, 10)+"?error=in-use", http.StatusSeeOther)
		default:
			a.serverError(w, r, err)
		}
		return
	}
	http.Redirect(w, r, "/services?notice=deleted", http.StatusSeeOther)
}

func (a *App) serviceFromRequest(w http.ResponseWriter, r *http.Request) (models.Service, bool) {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		a.notFound(w, r)
		return models.Service{}, false
	}
	service, err := a.store.GetService(r.Context(), id)
	if errors.Is(err, database.ErrNotFound) {
		a.notFound(w, r)
		return models.Service{}, false
	}
	if err != nil {
		a.serverError(w, r, err)
		return models.Service{}, false
	}
	return service, true
}

func parseServiceForm(r *http.Request) (models.Service, map[string]string, bool) {
	if err := r.ParseForm(); err != nil {
		return models.Service{}, map[string]string{"form": "The submitted form could not be read."}, false
	}
	input := validation.ServiceInput{
		Name: strings.TrimSpace(r.FormValue("name")), Description: strings.TrimSpace(r.FormValue("description")),
		OwnerTeam: strings.TrimSpace(r.FormValue("owner_team")), Status: models.ServiceStatus(r.FormValue("status")),
	}
	validationErrors := validation.ValidateService(input)
	service := models.Service{Name: input.Name, Description: input.Description, OwnerTeam: input.OwnerTeam, Status: input.Status}
	return service, validationErrors, len(validationErrors) == 0
}

func parseID(value string) (int64, error) {
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id < 1 {
		return 0, errors.New("invalid id")
	}
	return id, nil
}

func queryMessage(code string) string {
	if code == "in-use" {
		return "This service is linked to incident history and cannot be deleted."
	}
	return ""
}
