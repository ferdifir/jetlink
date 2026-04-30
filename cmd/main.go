package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/joho/godotenv"

	"github.com/ferdifir/jetlink/internal/config"
	"github.com/ferdifir/jetlink/internal/database"
	"github.com/ferdifir/jetlink/internal/handlers"
	"github.com/ferdifir/jetlink/internal/middleware"
	"github.com/ferdifir/jetlink/internal/repository"
	"github.com/ferdifir/jetlink/internal/services"
)

func main() {
	godotenv.Load()

	cfg := config.Load()

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	store := sessions.NewCookieStore([]byte(cfg.SessionSecret))
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}

	tenantRepo := repository.NewTenantRepo(db)
	customerRepo := repository.NewCustomerRepo(db)
	queueRepo := repository.NewQueueRepo(db)

	fonnte := services.NewFonnteClient(cfg.FonnteToken)
	authSvc := services.NewAuthService(customerRepo, fonnte, store)
	queueSvc := services.NewQueueService(queueRepo, tenantRepo)
	notifSvc := services.NewNotificationService(queueRepo, fonnte)
	queueSvc.SetNotificationService(notifSvc)

	customerHandler := handlers.NewCustomerHandler(authSvc, queueSvc)
	adminHandler := handlers.NewAdminHandler(authSvc, queueSvc, notifSvc, tenantRepo)
	superadminHandler := handlers.NewSuperadminHandler(authSvc, tenantRepo)

	r := mux.NewRouter()

	r.PathPrefix("/static/").Handler(
		http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))),
	)

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/superadmin", http.StatusFound)
	}).Methods("GET")

	sa := r.PathPrefix("/superadmin").Subrouter()
	sa.HandleFunc("/login", superadminHandler.ShowLogin).Methods("GET")
	sa.HandleFunc("/login", superadminHandler.ProcessLogin).Methods("POST")
	sa.HandleFunc("/logout", superadminHandler.Logout).Methods("POST")
	sa.HandleFunc("", superadminHandler.Dashboard).Methods("GET")
	sa.HandleFunc("/tenants", superadminHandler.CreateTenant).Methods("POST")
	sa.HandleFunc("/tenants/{id}", superadminHandler.UpdateTenant).Methods("POST")
	sa.HandleFunc("/tenants/{id}/delete", superadminHandler.DeleteTenant).Methods("POST")

	tenant := r.PathPrefix("/{slug}").Subrouter()
	tenant.Use(middleware.TenantMiddleware(tenantRepo))

	tenant.HandleFunc("", customerHandler.Index).Methods("GET")
	tenant.HandleFunc("/login", customerHandler.ShowLogin).Methods("GET")
	tenant.HandleFunc("/login", customerHandler.ProcessLogin).Methods("POST")
	tenant.HandleFunc("/verify", customerHandler.ShowVerify).Methods("GET")
	tenant.HandleFunc("/verify", customerHandler.ProcessVerify).Methods("POST")
	tenant.HandleFunc("/queue/take", customerHandler.TakeQueue).Methods("POST")
	tenant.HandleFunc("/logout", customerHandler.Logout).Methods("POST")
	tenant.HandleFunc("/api/status", customerHandler.APIStatus).Methods("GET")

	admin := tenant.PathPrefix("/admin").Subrouter()
	admin.HandleFunc("/login", adminHandler.ShowLogin).Methods("GET")
	admin.HandleFunc("/login", adminHandler.ProcessLogin).Methods("POST")
	admin.HandleFunc("/logout", adminHandler.Logout).Methods("POST")
	admin.HandleFunc("", adminHandler.Dashboard).Methods("GET")
	admin.HandleFunc("/queue", adminHandler.QueuePage).Methods("GET")
	admin.HandleFunc("/queue/open", adminHandler.OpenQueue).Methods("POST")
	admin.HandleFunc("/queue/close", adminHandler.CloseQueue).Methods("POST")
	admin.HandleFunc("/queue/next", adminHandler.NextQueue).Methods("POST")
	admin.HandleFunc("/queue/call/{num}", adminHandler.CallNumber).Methods("POST")
	admin.HandleFunc("/queue/skip/{num}", adminHandler.SkipNumber).Methods("POST")
	admin.HandleFunc("/queue/done/{num}", adminHandler.DoneNumber).Methods("POST")
	admin.HandleFunc("/notify/{ticketID}", adminHandler.SendNotification).Methods("POST")
	admin.HandleFunc("/settings", adminHandler.ShowSettings).Methods("GET")
	admin.HandleFunc("/settings", adminHandler.SaveSettings).Methods("POST")
	admin.HandleFunc("/api/queue", adminHandler.APIQueue).Methods("GET")

	log.Printf("Jetlink running on http://localhost:%s", cfg.AppPort)
	log.Fatal(http.ListenAndServe(":"+cfg.AppPort, r))
}
