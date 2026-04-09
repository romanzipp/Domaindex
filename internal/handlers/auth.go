package handlers

import (
	"net/http"

	"github.com/romanzipp/domaindex/internal/middleware"
	"github.com/romanzipp/domaindex/internal/models"
	"github.com/romanzipp/domaindex/internal/seeds"
	"golang.org/x/crypto/bcrypt"
)

func (b *Base) ShowLogin(w http.ResponseWriter, r *http.Request) {
	if middleware.UserFromContext(r.Context()) != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	b.render(w, r, "login.html", nil)
}

func (b *Base) Login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	var user models.User
	if err := b.db.Where("username = ?", username).First(&user).Error; err != nil {
		b.flashError(w, r, "Invalid username or password")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		b.flashError(w, r, "Invalid username or password")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	session, _ := b.store.Get(r, "session")
	session.Values["user_id"] = user.ID
	session.Save(r, w)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (b *Base) ShowRegister(w http.ResponseWriter, r *http.Request) {
	if !b.RegistrationEnabled {
		http.NotFound(w, r)
		return
	}
	if middleware.UserFromContext(r.Context()) != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	b.render(w, r, "register.html", nil)
}

func (b *Base) Register(w http.ResponseWriter, r *http.Request) {
	if !b.RegistrationEnabled {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	username := r.FormValue("username")
	email := r.FormValue("email")
	password := r.FormValue("password")

	if username == "" || email == "" || password == "" {
		b.flashError(w, r, "All fields are required")
		http.Redirect(w, r, "/register", http.StatusSeeOther)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	user := models.User{
		Username:     username,
		Email:        email,
		PasswordHash: string(hash),
	}

	if err := b.db.Create(&user).Error; err != nil {
		b.flashError(w, r, "Username or email already taken")
		http.Redirect(w, r, "/register", http.StatusSeeOther)
		return
	}

	seeds.ForUser(b.db, user.ID) //nolint:errcheck

	session, _ := b.store.Get(r, "session")
	session.Values["user_id"] = user.ID
	session.Save(r, w)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (b *Base) Logout(w http.ResponseWriter, r *http.Request) {
	session, _ := b.store.Get(r, "session")
	delete(session.Values, "user_id")
	session.Options.MaxAge = -1
	session.Save(r, w)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
