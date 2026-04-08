package middleware

import (
	"context"
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/romanzipp/domain-manager/internal/models"
	"gorm.io/gorm"
)

type contextKey string

const userContextKey contextKey = "user"

type AuthMiddleware struct {
	store *sessions.CookieStore
	db    *gorm.DB
}

func NewAuthMiddleware(store *sessions.CookieStore, db *gorm.DB) *AuthMiddleware {
	return &AuthMiddleware{store: store, db: db}
}

func (m *AuthMiddleware) Require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := m.userFromSession(r)
		if user == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		ctx := context.WithValue(r.Context(), userContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *AuthMiddleware) Load(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := m.userFromSession(r)
		if user != nil {
			ctx := context.WithValue(r.Context(), userContextKey, user)
			r = r.WithContext(ctx)
		}
		next.ServeHTTP(w, r)
	})
}

func (m *AuthMiddleware) userFromSession(r *http.Request) *models.User {
	session, err := m.store.Get(r, "session")
	if err != nil {
		return nil
	}
	userID, ok := session.Values["user_id"].(uint)
	if !ok || userID == 0 {
		return nil
	}
	var user models.User
	if err := m.db.First(&user, userID).Error; err != nil {
		return nil
	}
	return &user
}

func UserFromContext(ctx context.Context) *models.User {
	user, _ := ctx.Value(userContextKey).(*models.User)
	return user
}
