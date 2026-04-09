package handlers

import (
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/romanzipp/domain-manager/internal/middleware"
	"github.com/romanzipp/domain-manager/internal/models"
)

var validColors = func() map[string]bool {
	m := make(map[string]bool, len(models.TagColors))
	for _, c := range models.TagColors {
		m[c] = true
	}
	return m
}()

type TagsHandler struct {
	*Base
}

func NewTagsHandler(base *Base) *TagsHandler {
	return &TagsHandler{Base: base}
}

// AttachTag attaches an existing tag or creates a new one and attaches it.
func (h *TagsHandler) AttachTag(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	domain, ok := h.loadDomainForUser(w, r, user.ID)
	if !ok {
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	var tag models.Tag

	if tagID := r.FormValue("tag_id"); tagID != "" {
		if h.db.Where("id = ? AND user_id = ?", tagID, user.ID).First(&tag).Error != nil {
			h.flashError(w, r, "Tag not found")
			http.Redirect(w, r, "/domains/"+mux.Vars(r)["id"], http.StatusSeeOther)
			return
		}
	} else {
		name := strings.TrimSpace(r.FormValue("name"))
		color := r.FormValue("color")
		if name == "" {
			h.flashError(w, r, "Tag name is required")
			http.Redirect(w, r, "/domains/"+mux.Vars(r)["id"], http.StatusSeeOther)
			return
		}
		if !validColors[color] {
			color = "gray"
		}
		tag = models.Tag{UserID: user.ID, Name: name, Color: color}
		if err := h.db.Create(&tag).Error; err != nil {
			h.flashError(w, r, "Could not create tag")
			http.Redirect(w, r, "/domains/"+mux.Vars(r)["id"], http.StatusSeeOther)
			return
		}
	}

	h.db.Model(domain).Association("Tags").Append(&tag)
	http.Redirect(w, r, "/domains/"+mux.Vars(r)["id"], http.StatusSeeOther)
}

// DetachTag removes a tag from a domain.
func (h *TagsHandler) DetachTag(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	domain, ok := h.loadDomainForUser(w, r, user.ID)
	if !ok {
		return
	}

	var tag models.Tag
	if h.db.Where("id = ? AND user_id = ?", mux.Vars(r)["tag_id"], user.ID).First(&tag).Error != nil {
		http.NotFound(w, r)
		return
	}

	h.db.Model(domain).Association("Tags").Delete(&tag)
	http.Redirect(w, r, "/domains/"+mux.Vars(r)["id"], http.StatusSeeOther)
}

// DeleteTag deletes a tag globally (removes it from all domains).
func (h *TagsHandler) DeleteTag(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())

	var tag models.Tag
	if h.db.Where("id = ? AND user_id = ?", mux.Vars(r)["tag_id"], user.ID).First(&tag).Error != nil {
		http.NotFound(w, r)
		return
	}

	h.db.Model(&tag).Association("Domains").Clear()
	h.db.Delete(&tag)

	// Redirect back to the referring domain page if available.
	ref := r.Header.Get("Referer")
	if ref == "" {
		ref = "/domains"
	}
	http.Redirect(w, r, ref, http.StatusSeeOther)
}

func (h *TagsHandler) loadDomainForUser(w http.ResponseWriter, r *http.Request, userID uint) (*models.Domain, bool) {
	var domain models.Domain
	if err := h.db.Where("id = ? AND user_id = ?", mux.Vars(r)["id"], userID).First(&domain).Error; err != nil {
		http.NotFound(w, r)
		return nil, false
	}
	return &domain, true
}
