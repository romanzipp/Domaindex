package handlers

import (
	"net/http"

	"github.com/gorilla/sessions"
)

func setFlash(w http.ResponseWriter, r *http.Request, store *sessions.CookieStore, key, msg string) {
	session, _ := store.Get(r, "flash")
	session.AddFlash(msg, key)
	session.Save(r, w)
}

func getFlash(w http.ResponseWriter, r *http.Request, store *sessions.CookieStore, key string) []string {
	session, err := store.Get(r, "flash")
	if err != nil {
		return nil
	}
	flashes := session.Flashes(key)
	session.Save(r, w)
	msgs := make([]string, len(flashes))
	for i, f := range flashes {
		msgs[i], _ = f.(string)
	}
	return msgs
}
