package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/romanzipp/domaindex/internal/middleware"
	"github.com/romanzipp/domaindex/internal/models"
)

type RegistrarsHandler struct {
	*Base
}

func NewRegistrarsHandler(base *Base) *RegistrarsHandler {
	return &RegistrarsHandler{Base: base}
}

func (h *RegistrarsHandler) List(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	var registrars []models.Registrar
	h.db.Preload("Prices").Where("user_id = ?", user.ID).Order("name").Find(&registrars)
	h.render(w, r, "registrars/list.html", registrars)
}

func (h *RegistrarsHandler) ShowAdd(w http.ResponseWriter, r *http.Request) {
	h.render(w, r, "registrars/add.html", nil)
}

func (h *RegistrarsHandler) Add(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		h.flashError(w, r, "Name is required")
		http.Redirect(w, r, "/registrars/add", http.StatusSeeOther)
		return
	}

	registrar := models.Registrar{
		UserID:   user.ID,
		Name:     name,
		URL:      r.FormValue("url"),
		IanaID:   strings.TrimSpace(r.FormValue("iana_id")),
		Notes:    r.FormValue("notes"),
		Currency: r.FormValue("currency"),
	}
	if registrar.Currency == "" {
		registrar.Currency = "USD"
	}

	if err := h.db.Create(&registrar).Error; err != nil {
		h.flashError(w, r, "Failed to create registrar")
		errRedirect := "/registrars/add"
		if next := r.FormValue("next"); next != "" {
			errRedirect = next
		}
		http.Redirect(w, r, errRedirect, http.StatusSeeOther)
		return
	}

	h.flashSuccess(w, r, "Registrar added")
	if next := r.FormValue("next"); next != "" {
		http.Redirect(w, r, next, http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/registrars/"+idStr(registrar.ID), http.StatusSeeOther)
}

func (h *RegistrarsHandler) Show(w http.ResponseWriter, r *http.Request) {
	reg, ok := h.loadRegistrar(w, r)
	if !ok {
		return
	}
	var prices []models.Price
	h.db.Where("registrar_id = ?", reg.ID).Find(&prices)
	h.render(w, r, "registrars/detail.html", map[string]any{
		"Registrar": reg,
		"Prices":    prices,
	})
}

func (h *RegistrarsHandler) Update(w http.ResponseWriter, r *http.Request) {
	reg, ok := h.loadRegistrar(w, r)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	reg.Name = strings.TrimSpace(r.FormValue("name"))
	reg.URL = r.FormValue("url")
	reg.IanaID = strings.TrimSpace(r.FormValue("iana_id"))
	reg.Notes = r.FormValue("notes")
	reg.Currency = r.FormValue("currency")

	h.db.Save(reg)
	h.flashSuccess(w, r, "Registrar updated")
	http.Redirect(w, r, "/registrars/"+mux.Vars(r)["id"], http.StatusSeeOther)
}

func (h *RegistrarsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	reg, ok := h.loadRegistrar(w, r)
	if !ok {
		return
	}
	h.db.Where("registrar_id = ?", reg.ID).Delete(&models.Price{})
	h.db.Delete(reg)
	h.flashSuccess(w, r, "Registrar deleted")
	http.Redirect(w, r, "/registrars", http.StatusSeeOther)
}

func (h *RegistrarsHandler) AddPrice(w http.ResponseWriter, r *http.Request) {
	reg, ok := h.loadRegistrar(w, r)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	price := models.Price{
		RegistrarID:     &reg.ID,
		TLD:             r.FormValue("tld"),
		InitialPerYear:  parseFloat(r.FormValue("initial_per_year")),
		RenewPerYear:    parseFloat(r.FormValue("renew_per_year")),
		Transfer: parseFloat(r.FormValue("transfer")),
		PrivacyPerYear:  parseFloat(r.FormValue("privacy_per_year")),
		MiscPerYear:     parseFloat(r.FormValue("misc_per_year")),
	}

	h.db.Create(&price)
	h.flashSuccess(w, r, "Price added")
	http.Redirect(w, r, "/registrars/"+mux.Vars(r)["id"], http.StatusSeeOther)
}

func (h *RegistrarsHandler) DeletePrice(w http.ResponseWriter, r *http.Request) {
	reg, ok := h.loadRegistrar(w, r)
	if !ok {
		return
	}
	priceID := mux.Vars(r)["price_id"]
	h.db.Where("id = ? AND registrar_id = ?", priceID, reg.ID).Delete(&models.Price{})
	h.flashSuccess(w, r, "Price deleted")
	http.Redirect(w, r, "/registrars/"+mux.Vars(r)["id"], http.StatusSeeOther)
}

func (h *RegistrarsHandler) loadRegistrar(w http.ResponseWriter, r *http.Request) (*models.Registrar, bool) {
	user := middleware.UserFromContext(r.Context())
	id := mux.Vars(r)["id"]
	var reg models.Registrar
	if err := h.db.Where("id = ? AND user_id = ?", id, user.ID).First(&reg).Error; err != nil {
		http.NotFound(w, r)
		return nil, false
	}
	return &reg, true
}

func idStr(id uint) string {
	return fmt.Sprintf("%d", id)
}
