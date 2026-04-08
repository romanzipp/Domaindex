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
	db        *gorm.DB
	store     *sessions.CookieStore
	templates *template.Template
}

func NewBase(db *gorm.DB, store *sessions.CookieStore, templateFS fs.FS) (*Base, error) {
	tmpl, err := template.New("").Funcs(templateFuncs()).ParseFS(templateFS, "templates/**/*.html", "templates/*.html")
	if err != nil {
		return nil, err
	}
	return &Base{db: db, store: store, templates: tmpl}, nil
}

type PageData struct {
	User          *models.User
	Data          any
	Errors        []string
	FlashSuccess  []string
	FlashError    []string
}

func (b *Base) render(w http.ResponseWriter, r *http.Request, name string, data any) {
	user := middleware.UserFromContext(r.Context())
	pd := PageData{
		User:         user,
		Data:         data,
		FlashSuccess: getFlash(w, r, b.store, "success"),
		FlashError:   getFlash(w, r, b.store, "error"),
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := b.templates.ExecuteTemplate(w, name, pd); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (b *Base) flashSuccess(w http.ResponseWriter, r *http.Request, msg string) {
	setFlash(w, r, b.store, "success", msg)
}

func (b *Base) flashError(w http.ResponseWriter, r *http.Request, msg string) {
	setFlash(w, r, b.store, "error", msg)
}
