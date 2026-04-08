package handlers

import (
	"html/template"
	"io/fs"
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/romanzipp/domain-manager/internal/middleware"
	"github.com/romanzipp/domain-manager/internal/models"
	"gorm.io/gorm"
)

type Base struct {
	db                  *gorm.DB
	store               *sessions.CookieStore
	templateFS          fs.FS
	RegistrationEnabled bool
}

func NewBase(db *gorm.DB, store *sessions.CookieStore, templateFS fs.FS, registrationEnabled bool) *Base {
	return &Base{db: db, store: store, templateFS: templateFS, RegistrationEnabled: registrationEnabled}
}

type PageData struct {
	User         *models.User
	Data         any
	FlashSuccess []string
	FlashError   []string
}

func (b *Base) render(w http.ResponseWriter, r *http.Request, page string, data any, partials ...string) {
	files := []string{"layout/base.html", "layout/nav.html", "pages/" + page}
	for _, p := range partials {
		files = append(files, "pages/"+p)
	}
	tmpl, err := template.New("").Funcs(templateFuncs()).ParseFS(b.templateFS, files...)
	if err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	user := middleware.UserFromContext(r.Context())
	pd := PageData{
		User:         user,
		Data:         data,
		FlashSuccess: getFlash(w, r, b.store, "success"),
		FlashError:   getFlash(w, r, b.store, "error"),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "base.html", pd); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (b *Base) flashSuccess(w http.ResponseWriter, r *http.Request, msg string) {
	setFlash(w, r, b.store, "success", msg)
}

func (b *Base) flashError(w http.ResponseWriter, r *http.Request, msg string) {
	setFlash(w, r, b.store, "error", msg)
}
