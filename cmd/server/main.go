package main

import (
	"fmt"
	"io/fs"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/romanzipp/domain-manager/assets"
	"github.com/romanzipp/domain-manager/internal/config"
	"github.com/romanzipp/domain-manager/internal/db"
	"github.com/romanzipp/domain-manager/internal/handlers"
	"github.com/romanzipp/domain-manager/internal/jobs"
	"github.com/romanzipp/domain-manager/internal/middleware"
	"github.com/romanzipp/domain-manager/internal/services"
)

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

	base := handlers.NewBase(database, store, templateSub, cfg.RegistrationEnabled)

	whoisSvc := services.NewWhoisService(database)
	priceSvc := services.NewPriceService(database)
	notifSvc := services.NewNotificationService(database, cfg.AppriseURL, cfg.AppriseKey)

	domainsH := handlers.NewDomainsHandler(base, whoisSvc, priceSvc)
	registrarsH := handlers.NewRegistrarsHandler(base)
	notificationsH := handlers.NewNotificationsHandler(base)

	worker := jobs.NewWorker(database, whoisSvc, notifSvc, cfg.WhoisRefreshInterval)
	worker.Start()

	auth := middleware.NewAuthMiddleware(store, database)

	r := mux.NewRouter()

	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.FS(staticSub))))

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
	protected.HandleFunc("/domains/{id}/price", domainsH.SavePriceOverride).Methods("POST")
	protected.HandleFunc("/domains/{id}/price/delete", domainsH.DeletePriceOverride).Methods("POST")

	protected.HandleFunc("/registrars", registrarsH.List).Methods("GET")
	protected.HandleFunc("/registrars", registrarsH.Add).Methods("POST")
	protected.HandleFunc("/registrars/add", registrarsH.ShowAdd).Methods("GET")
	protected.HandleFunc("/registrars/{id}", registrarsH.Show).Methods("GET")
	protected.HandleFunc("/registrars/{id}", registrarsH.Update).Methods("POST")
	protected.HandleFunc("/registrars/{id}/delete", registrarsH.Delete).Methods("POST")
	protected.HandleFunc("/registrars/{id}/prices", registrarsH.AddPrice).Methods("POST")
	protected.HandleFunc("/registrars/{id}/prices/{price_id}/delete", registrarsH.DeletePrice).Methods("POST")

	protected.HandleFunc("/notifications", notificationsH.List).Methods("GET")

	addr := fmt.Sprintf("%s:%s", cfg.AppHost, cfg.AppPort)
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("server: %v", err)
	}
}
