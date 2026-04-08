package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/romanzipp/domain-manager/internal/middleware"
	"github.com/romanzipp/domain-manager/internal/models"
	"github.com/romanzipp/domain-manager/internal/services"
)

type DomainsHandler struct {
	*Base
	whois    *services.WhoisService
	price    *services.PriceService
	currency *services.CurrencyService
}

func NewDomainsHandler(base *Base, whois *services.WhoisService, price *services.PriceService, currency *services.CurrencyService) *DomainsHandler {
	return &DomainsHandler{Base: base, whois: whois, price: price, currency: currency}
}

type DomainRow struct {
	Domain          *models.Domain
	Price           *models.Price
	YearlyCost      *float64
	PriceCurrency   string
}

type DomainsListData struct {
	Rows            []DomainRow
	Sort            string
	Dir             string
	Registrars      []models.Registrar
	DefaultCurrency string
}

func (h *DomainsHandler) List(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())

	sort := r.URL.Query().Get("sort")
	dir := r.URL.Query().Get("dir")
	if dir != "asc" && dir != "desc" {
		dir = "asc"
	}

	allowedSorts := map[string]string{
		"name":            "domains.name",
		"expiration_date": "domains.expiration_date",
		"registrar":       `"Registrar"."name"`,
		"auto_renewed":    "domains.auto_renewed",
		"wishlisted":      "domains.wishlisted",
		"created_at":      "domains.created_at",
	}

	orderCol, ok := allowedSorts[sort]
	if !ok {
		sort = "name"
		orderCol = "domains.name"
	}

	var domains []models.Domain
	q := h.db.Joins("Registrar").
		Where("domains.user_id = ?", user.ID).
		Order(orderCol + " " + dir)

	if err := q.Find(&domains).Error; err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	targetCurrency := user.DefaultCurrency
	if targetCurrency == "" {
		targetCurrency = "USD"
	}

	rows := make([]DomainRow, len(domains))
	for i := range domains {
		d := &domains[i]
		price := h.price.ComputedPrice(d)

		row := DomainRow{Domain: d, Price: price}

		if price != nil {
			yearly := price.RenewPerYear + price.PrivacyPerYear + price.MiscPerYear
			sourceCurrency := targetCurrency
			if d.Registrar != nil && d.Registrar.Currency != "" {
				sourceCurrency = d.Registrar.Currency
			}
			converted := h.currency.Convert(yearly, sourceCurrency, targetCurrency)
			row.YearlyCost = &converted
			row.PriceCurrency = targetCurrency
		}

		rows[i] = row
	}

	var registrars []models.Registrar
	h.db.Where("user_id = ?", user.ID).Find(&registrars)

	h.render(w, r, "domains/list.html", DomainsListData{
		Rows:            rows,
		Sort:            sort,
		Dir:             dir,
		Registrars:      registrars,
		DefaultCurrency: targetCurrency,
	})
}

func (h *DomainsHandler) ShowAdd(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	var registrars []models.Registrar
	h.db.Where("user_id = ?", user.ID).Find(&registrars)
	h.render(w, r, "domains/add.html", map[string]any{"Registrars": registrars}, "registrars/_fields.html")
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
	h.render(w, r, "domains/bulk.html", map[string]any{"Registrars": registrars}, "registrars/_fields.html")
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
	var errs []string

	for _, line := range lines {
		name := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(line), " ", ""))
		if name == "" {
			continue
		}
		var existing models.Domain
		if h.db.Where("user_id = ? AND name = ?", user.ID, name).First(&existing).Error == nil {
			errs = append(errs, name+": already exists")
			continue
		}
		domain := h.buildDomain(r, user.ID, name)
		_, whoisResult, _ := h.whois.UpdateDomain(domain)
		if r.FormValue("registrar_id") == "whois" && whoisResult != nil {
			domain.RegistrarID = h.registrarFromWhois(whoisResult, user.ID)
		}
		if err := h.db.Create(domain).Error; err != nil {
			errs = append(errs, name+": could not save")
			continue
		}
		added++
	}

	for _, e := range errs {
		h.flashError(w, r, e)
	}
	if added > 0 {
		h.flashSuccess(w, r, fmt.Sprintf("Added %d domain(s)", added))
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

	if _, _, err := h.whois.UpdateDomain(domain); err != nil {
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
	price.Transfer = parseFloat(r.FormValue("transfer"))
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
	d.RegistrarID = h.resolveRegistrarID(r, userID)
	return d
}

// resolveRegistrarID creates a new registrar if new_registrar_name is filled,
// otherwise uses the selected registrar_id from the form.
func (h *DomainsHandler) resolveRegistrarID(r *http.Request, userID uint) *uint {
	if newName := strings.TrimSpace(r.FormValue("new_registrar_name")); newName != "" {
		currency := strings.TrimSpace(r.FormValue("new_registrar_currency"))
		if currency == "" {
			currency = "USD"
		}
		reg := models.Registrar{
			UserID:   userID,
			Name:     newName,
			URL:      r.FormValue("new_registrar_url"),
			Currency: currency,
		}
		if h.db.Create(&reg).Error == nil {
			return &reg.ID
		}
		return nil
	}
	// "whois" is a special sentinel — registrar will be resolved after WHOIS fetch
	if regID := r.FormValue("registrar_id"); regID != "" && regID != "0" && regID != "whois" {
		var reg models.Registrar
		if h.db.Where("id = ? AND user_id = ?", regID, userID).First(&reg).Error == nil {
			return &reg.ID
		}
	}
	return nil
}

// registrarFromWhois finds or creates a registrar using WHOIS data.
// Looks up by IANA ID first (globally unique); falls back to name if no IANA ID.
func (h *DomainsHandler) registrarFromWhois(result *services.WhoisResult, userID uint) *uint {
	if result.RegistrarName == "" {
		return nil
	}
	var reg models.Registrar
	if result.RegistrarIanaID != "" {
		if h.db.Where("user_id = ? AND iana_id = ?", userID, result.RegistrarIanaID).First(&reg).Error == nil {
			return &reg.ID
		}
	}
	reg = models.Registrar{
		UserID:  userID,
		Name:    result.RegistrarName,
		IanaID:  result.RegistrarIanaID,
		URL:     result.RegistrarURL,
		Currency: "USD",
	}
	if h.db.Create(&reg).Error != nil {
		return nil
	}
	return &reg.ID
}

func (h *DomainsHandler) fetchAndSaveDomain(w http.ResponseWriter, r *http.Request, domain *models.Domain, errRedirect string) {
	var existing models.Domain
	if h.db.Where("user_id = ? AND name = ?", domain.UserID, domain.Name).First(&existing).Error == nil {
		h.flashError(w, r, fmt.Sprintf("%s already exists", domain.Name))
		http.Redirect(w, r, errRedirect, http.StatusSeeOther)
		return
	}

	_, whoisResult, _ := h.whois.UpdateDomain(domain) // best-effort
	if r.FormValue("registrar_id") == "whois" && whoisResult != nil {
		domain.RegistrarID = h.registrarFromWhois(whoisResult, domain.UserID)
	}

	if err := h.db.Create(domain).Error; err != nil {
		h.flashError(w, r, fmt.Sprintf("Could not save %s", domain.Name))
		http.Redirect(w, r, errRedirect, http.StatusSeeOther)
		return
	}

	h.flashSuccess(w, r, "Domain added")
	http.Redirect(w, r, "/domains", http.StatusSeeOther)
}
