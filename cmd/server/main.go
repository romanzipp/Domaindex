package main

import (
	"fmt"
	"io/fs"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/romanzipp/domaindex/assets"
	"github.com/romanzipp/domaindex/internal/config"
	"github.com/romanzipp/domaindex/internal/db"
	"github.com/romanzipp/domaindex/internal/handlers"
	"github.com/romanzipp/domaindex/internal/jobs"
	"github.com/romanzipp/domaindex/internal/middleware"
	"github.com/romanzipp/domaindex/internal/services"
)

// version is injected at build time via -ldflags "-X main.version=x.y.z".
// It is empty in development builds.
var version string

func main() {
	cfg := config.Load()

	database, err := db.Connect(cfg)
	if err != nil {
		log.Fatalf("database: %v", err)
	}

	store := sessions.NewCookieStore([]byte(cfg.AppSecret))
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   0,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}

	templateSub, err := fs.Sub(assets.Templates, "templates")
	if err != nil {
		log.Fatalf("template fs: %v", err)
	}

	staticSub, err := fs.Sub(assets.Static, "static")
	if err != nil {
		log.Fatalf("static fs: %v", err)
	}

	base := handlers.NewBase(database, store, templateSub, cfg.RegistrationEnabled, version)

	whoisSvc := services.NewWhoisService(database)
	techSvc := services.NewTechService(database)
	priceSvc := services.NewPriceService(database)
	notifSvc := services.NewNotificationService(database, cfg.AppriseURL, cfg.AppriseKey, cfg.AppriseTag)
	currencySvc := services.NewCurrencyService()

	registrarsH := handlers.NewRegistrarsHandler(base)
	notificationsH := handlers.NewNotificationsHandler(base, notifSvc)
	tagsH := handlers.NewTagsHandler(base)

	worker := jobs.NewWorker(database, whoisSvc, techSvc, notifSvc, cfg.WhoisRefreshInterval)
	worker.Start()

	domainsH := handlers.NewDomainsHandler(base, whoisSvc, techSvc, priceSvc, currencySvc, worker)

	auth := middleware.NewAuthMiddleware(store, database)

	r := mux.NewRouter()

	staticHandler := http.StripPrefix("/static/", http.FileServer(http.FS(staticSub)))
	r.PathPrefix("/static/").Handler(staticCacheHandler(staticHandler, version))

	r.Handle("/login", auth.Load(http.HandlerFunc(base.ShowLogin))).Methods("GET")
	r.Handle("/login", auth.Load(http.HandlerFunc(base.Login))).Methods("POST")
	r.Handle("/register", auth.Load(http.HandlerFunc(base.ShowRegister))).Methods("GET")
	r.Handle("/register", auth.Load(http.HandlerFunc(base.Register))).Methods("POST")
	r.Handle("/logout", auth.Require(http.HandlerFunc(base.Logout))).Methods("POST")

	protected := r.NewRoute().Subrouter()
	protected.Use(auth.Require)

	protected.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/domains", http.StatusSeeOther)
	}).Methods("GET")

	protected.HandleFunc("/domains", domainsH.List).Methods("GET")
	protected.HandleFunc("/domains", domainsH.Add).Methods("POST")
	protected.HandleFunc("/domains/add", domainsH.ShowAdd).Methods("GET")
	protected.HandleFunc("/domains/bulk", domainsH.ShowBulk).Methods("GET")
	protected.HandleFunc("/domains/bulk", domainsH.Bulk).Methods("POST")
	protected.HandleFunc("/domains/{id}", domainsH.Show).Methods("GET")
	protected.HandleFunc("/domains/{id}", domainsH.Update).Methods("POST")
	protected.HandleFunc("/domains/{id}/delete", domainsH.Delete).Methods("POST")
	protected.HandleFunc("/domains/{id}/refresh", domainsH.RefreshWhois).Methods("POST")
	protected.HandleFunc("/domains/{id}/refresh-tech", domainsH.RefreshTech).Methods("POST")
	protected.HandleFunc("/domains/{id}/price", domainsH.SavePriceOverride).Methods("POST")
	protected.HandleFunc("/domains/{id}/price/delete", domainsH.DeletePriceOverride).Methods("POST")
	protected.HandleFunc("/domains/{id}/tags", tagsH.AttachTag).Methods("POST")
	protected.HandleFunc("/domains/{id}/tags/{tag_id}/detach", tagsH.DetachTag).Methods("POST")
	protected.HandleFunc("/tags/{tag_id}/delete", tagsH.DeleteTag).Methods("POST")

	protected.HandleFunc("/registrars", registrarsH.List).Methods("GET")
	protected.HandleFunc("/registrars", registrarsH.Add).Methods("POST")
	protected.HandleFunc("/registrars/add", registrarsH.ShowAdd).Methods("GET")
	protected.HandleFunc("/registrars/{id}", registrarsH.Show).Methods("GET")
	protected.HandleFunc("/registrars/{id}", registrarsH.Update).Methods("POST")
	protected.HandleFunc("/registrars/{id}/delete", registrarsH.Delete).Methods("POST")
	protected.HandleFunc("/registrars/{id}/prices", registrarsH.AddPrice).Methods("POST")
	protected.HandleFunc("/registrars/{id}/prices/{price_id}/delete", registrarsH.DeletePrice).Methods("POST")

	protected.HandleFunc("/notifications", notificationsH.List).Methods("GET")
	protected.HandleFunc("/notifications/test", notificationsH.SendTest).Methods("POST")

	protected.HandleFunc("/profile", base.ShowProfile).Methods("GET")
	protected.HandleFunc("/profile", base.UpdateProfile).Methods("POST")
	protected.HandleFunc("/profile/password", base.UpdatePassword).Methods("POST")

	addr := fmt.Sprintf("%s:%s", cfg.AppHost, cfg.AppPort)
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("server: %v", err)
	}
}

// staticCacheHandler adds long-lived cache headers when a version is set (production).
// In dev (version == ""), no cache headers are added so asset changes are picked up immediately.
func staticCacheHandler(h http.Handler, version string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if version != "" {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		h.ServeHTTP(w, r)
	})
}
