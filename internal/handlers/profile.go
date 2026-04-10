package handlers

import (
	"net/http"
	"strings"

	"github.com/romanzipp/domaindex/internal/middleware"
	"golang.org/x/crypto/bcrypt"
)

func (b *Base) ShowProfile(w http.ResponseWriter, r *http.Request) {
	b.render(w, r, "profile.html", nil)
}

func (b *Base) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	username := strings.TrimSpace(r.FormValue("username"))
	email := strings.TrimSpace(r.FormValue("email"))
	currency := strings.ToUpper(strings.TrimSpace(r.FormValue("default_currency")))
	techInfoEnabled := r.FormValue("tech_info_enabled") == "on"

	if username == "" || email == "" {
		b.flashError(w, r, "Username and email are required")
		http.Redirect(w, r, "/profile", http.StatusSeeOther)
		return
	}
	if currency == "" {
		currency = "USD"
	}

	user.Username = username
	user.Email = email
	user.DefaultCurrency = currency
	user.TechInfoEnabled = techInfoEnabled

	if err := b.db.Save(user).Error; err != nil {
		b.flashError(w, r, "Failed to update profile (username or email already taken)")
		http.Redirect(w, r, "/profile", http.StatusSeeOther)
		return
	}

	b.flashSuccess(w, r, "Profile updated")
	http.Redirect(w, r, "/profile", http.StatusSeeOther)
}

func (b *Base) UpdatePassword(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	current := r.FormValue("current_password")
	newPw := r.FormValue("new_password")
	confirm := r.FormValue("confirm_password")

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(current)); err != nil {
		b.flashError(w, r, "Current password is incorrect")
		http.Redirect(w, r, "/profile", http.StatusSeeOther)
		return
	}

	if newPw == "" || newPw != confirm {
		b.flashError(w, r, "New passwords do not match or are empty")
		http.Redirect(w, r, "/profile", http.StatusSeeOther)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPw), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	user.PasswordHash = string(hash)
	b.db.Save(user)

	b.flashSuccess(w, r, "Password updated")
	http.Redirect(w, r, "/profile", http.StatusSeeOther)
}
