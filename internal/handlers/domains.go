package handlers

import (
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/romanzipp/domain-manager/internal/middleware"
	"github.com/romanzipp/domain-manager/internal/models"
	"github.com/romanzipp/domain-manager/internal/services"
)

type DomainsHandler struct {
	*Base
	whois *services.WhoisService
	price *services.PriceService
}

func NewDomainsHandler(base *Base, whois *services.WhoisService, price *services.PriceService) *DomainsHandler {
	return &DomainsHandler{Base: base, whois: whois, price: price}
}

type DomainRow struct {
	Domain *models.Domain
	Price  *models.Price
}

type DomainsListData struct {
	Rows      []DomainRow
	Sort      string
	Dir       string
	Registrars []models.Registrar
}

func (h *DomainsHandler) List(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())

	sort := r.URL.Query().Get("sort")
	dir := r.URL.Query().Get("dir")
	if dir != "asc" && dir != "desc" {
		dir = "asc"
	}

	allowedSorts := map[string]string{
		"name":            "name",
		"expiration_date": "expiration_date",
		"registrar":       "registrars.name",
		"auto_renewed":    "auto_renewed",
		"wishlisted":      "wishlisted",
		"created_at":      "domains.created_at",
	}

	orderCol, ok := allowedSorts[sort]
	if !ok {
		sort = "name"
		orderCol = "name"
	}

	var domains []models.Domain
	q := h.db.Preload("Registrar").Where("domains.user_id = ?", user.ID).
		Joins("LEFT JOIN registrars ON registrars.id = domains.registrar_id").
		Order(orderCol + " " + dir)

	if err := q.Find(&domains).Error; err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	rows := make([]DomainRow, len(domains))
	for i := range domains {
		rows[i] = DomainRow{
			Domain: &domains[i],
			Price:  h.price.ComputedPrice(&domains[i]),
		}
	}

	var registrars []models.Registrar
	h.db.Where("user_id = ?", user.ID).Find(&registrars)

	h.render(w, r, "domains/list.html", DomainsListData{
		Rows:       rows,
		Sort:       sort,
		Dir:        dir,
		Registrars: registrars,
	})
}

func (h *DomainsHandler) ShowAdd(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	var registrars []models.Registrar
	h.db.Where("user_id = ?", user.ID).Find(&registrars)
	h.render(w, r, "domains/add.html", map[string]any{"Registrars": registrars})
}

func (h *DomainsHandler) Add(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	name := strings.ToLower(strings.TrimSpace(r.FormValue("name")))
	if name == "" {
		h.flashError(w, r, "Domain name is required")
		http.Redirect(w, r, "/domains/add", http.StatusSeeOther)
		return
	}

	domain := h.buildDomain(r, user.ID, name)
	h.fetchAndSaveDomain(w, r, domain, "/domains/add")
}

func (h *DomainsHandler) ShowBulk(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	var registrars []models.Registrar
	h.db.Where("user_id = ?", user.ID).Find(&registrars)
	h.render(w, r, "domains/bulk.html", map[string]any{"Registrars": registrars})
}

func (h *DomainsHandler) Bulk(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	raw := r.FormValue("domains")
	lines := strings.Split(raw, "\n")

	added := 0
	var errors []string

	for _, line := range lines {
		name := strings.ToLower(strings.TrimSpace(line))
		if name == "" {
			continue
		}
		domain := h.buildDomain(r, user.ID, name)
		_, err := h.whois.UpdateDomain(domain)
		if err != nil {
			errors = append(errors, name+": WHOIS fetch failed")
		}
		if err := h.db.Create(domain).Error; err != nil {
			errors = append(errors, name+": already exists or db error")
			continue
		}
		added++
	}

	if len(errors) > 0 {
		for _, e := range errors {
			h.flashError(w, r, e)
		}
	}
	h.flashSuccess(w, r, strings.Repeat(".", 0)+strings.TrimSuffix(strings.Repeat("added, ", added), ", "))
	if added > 0 {
		h.flashSuccess(w, r, "Added "+strings.TrimSpace(strings.Repeat("1 ", added))+" domain(s)")
	}
	http.Redirect(w, r, "/domains", http.StatusSeeOther)
}

func (h *DomainsHandler) Show(w http.ResponseWriter, r *http.Request) {
	domain, ok := h.loadDomain(w, r)
	if !ok {
		return
	}
	price := h.price.ComputedPrice(domain)

	var registrars []models.Registrar
	user := middleware.UserFromContext(r.Context())
	h.db.Where("user_id = ?", user.ID).Find(&registrars)

	h.render(w, r, "domains/detail.html", map[string]any{
		"Domain":     domain,
		"Price":      price,
		"Registrars": registrars,
	})
}

func (h *DomainsHandler) Update(w http.ResponseWriter, r *http.Request) {
	domain, ok := h.loadDomain(w, r)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	domain.AutoRenewed = r.FormValue("auto_renewed") == "on"
	domain.Wishlisted = r.FormValue("wishlisted") == "on"

	if regID := r.FormValue("registrar_id"); regID != "" && regID != "0" {
		var reg models.Registrar
		if err := h.db.First(&reg, regID).Error; err == nil {
			domain.RegistrarID = &reg.ID
		}
	} else {
		domain.RegistrarID = nil
	}

	if err := h.db.Save(domain).Error; err != nil {
		h.flashError(w, r, "Failed to update domain")
	} else {
		h.flashSuccess(w, r, "Domain updated")
	}

	http.Redirect(w, r, "/domains/"+mux.Vars(r)["id"], http.StatusSeeOther)
}

func (h *DomainsHandler) RefreshWhois(w http.ResponseWriter, r *http.Request) {
	domain, ok := h.loadDomain(w, r)
	if !ok {
		return
	}

	if _, err := h.whois.UpdateDomain(domain); err != nil {
		h.flashError(w, r, "WHOIS fetch failed: "+err.Error())
	} else {
		h.db.Save(domain)
		h.flashSuccess(w, r, "WHOIS info refreshed")
	}

	http.Redirect(w, r, "/domains/"+mux.Vars(r)["id"], http.StatusSeeOther)
}

func (h *DomainsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	domain, ok := h.loadDomain(w, r)
	if !ok {
		return
	}
	h.db.Where("domain_id = ?", domain.ID).Delete(&models.Price{})
	h.db.Where("domain_id = ?", domain.ID).Delete(&models.Notification{})
	h.db.Delete(domain)
	h.flashSuccess(w, r, "Domain deleted")
	http.Redirect(w, r, "/domains", http.StatusSeeOther)
}

func (h *DomainsHandler) SavePriceOverride(w http.ResponseWriter, r *http.Request) {
	domain, ok := h.loadDomain(w, r)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	price := models.Price{DomainID: &domain.ID}
	h.db.Where("domain_id = ?", domain.ID).FirstOrInit(&price)
	price.InitialPerYear = parseFloat(r.FormValue("initial_per_year"))
	price.RenewPerYear = parseFloat(r.FormValue("renew_per_year"))
	price.TransferPerYear = parseFloat(r.FormValue("transfer_per_year"))
	price.PrivacyPerYear = parseFloat(r.FormValue("privacy_per_year"))
	price.MiscPerYear = parseFloat(r.FormValue("misc_per_year"))

	h.db.Save(&price)
	h.flashSuccess(w, r, "Price override saved")
	http.Redirect(w, r, "/domains/"+mux.Vars(r)["id"], http.StatusSeeOther)
}

func (h *DomainsHandler) DeletePriceOverride(w http.ResponseWriter, r *http.Request) {
	domain, ok := h.loadDomain(w, r)
	if !ok {
		return
	}
	h.db.Where("domain_id = ?", domain.ID).Delete(&models.Price{})
	h.flashSuccess(w, r, "Price override removed")
	http.Redirect(w, r, "/domains/"+mux.Vars(r)["id"], http.StatusSeeOther)
}

func (h *DomainsHandler) loadDomain(w http.ResponseWriter, r *http.Request) (*models.Domain, bool) {
	user := middleware.UserFromContext(r.Context())
	id := mux.Vars(r)["id"]
	var domain models.Domain
	if err := h.db.Preload("Registrar").Where("id = ? AND user_id = ?", id, user.ID).First(&domain).Error; err != nil {
		http.NotFound(w, r)
		return nil, false
	}
	return &domain, true
}

func (h *DomainsHandler) buildDomain(r *http.Request, userID uint, name string) *models.Domain {
	d := &models.Domain{
		UserID:      userID,
		Name:        name,
		TLD:         services.ExtractTLD(name),
		AutoRenewed: r.FormValue("auto_renewed") == "on",
		Wishlisted:  r.FormValue("wishlisted") == "on",
	}
	if regID := r.FormValue("registrar_id"); regID != "" && regID != "0" {
		var reg models.Registrar
		if h.db.First(&reg, regID).Error == nil {
			d.RegistrarID = &reg.ID
		}
	}
	return d
}

func (h *DomainsHandler) fetchAndSaveDomain(w http.ResponseWriter, r *http.Request, domain *models.Domain, errRedirect string) {
	h.whois.UpdateDomain(domain) // best-effort, don't fail on whois error

	if err := h.db.Create(domain).Error; err != nil {
		h.flashError(w, r, "Domain already exists or could not be saved")
		http.Redirect(w, r, errRedirect, http.StatusSeeOther)
		return
	}

	h.flashSuccess(w, r, "Domain added")
	http.Redirect(w, r, "/domains", http.StatusSeeOther)
}
