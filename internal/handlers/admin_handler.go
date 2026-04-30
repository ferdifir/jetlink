package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"

	"github.com/ferdifir/jetlink/internal/middleware"
	"github.com/ferdifir/jetlink/internal/models"
	"github.com/ferdifir/jetlink/internal/repository"
	"github.com/ferdifir/jetlink/internal/services"
)

type AdminHandler struct {
	authSvc    *services.AuthService
	queueSvc   *services.QueueService
	notifSvc   *services.NotificationService
	tenantRepo *repository.TenantRepo
}

func NewAdminHandler(
	authSvc *services.AuthService,
	queueSvc *services.QueueService,
	notifSvc *services.NotificationService,
	tenantRepo *repository.TenantRepo,
) *AdminHandler {
	return &AdminHandler{
		authSvc:    authSvc,
		queueSvc:   queueSvc,
		notifSvc:   notifSvc,
		tenantRepo: tenantRepo,
	}
}

func (h *AdminHandler) tenant(r *http.Request) *models.Tenant {
	return r.Context().Value(middleware.TenantKey).(*models.Tenant)
}

func (h *AdminHandler) requireAuth(w http.ResponseWriter, r *http.Request) (int, bool) {
	tenantID, ok := h.authSvc.GetAdminSession(r)
	if !ok {
		tenant := h.tenant(r)
		redirectf(w, r, "/"+tenant.Slug+"/admin/login")
		return 0, false
	}

	tenant := h.tenant(r)
	if tenantID != tenant.ID {
		redirectf(w, r, "/"+tenant.Slug+"/admin/login")
		return 0, false
	}
	return tenantID, true
}

func (h *AdminHandler) ShowLogin(w http.ResponseWriter, r *http.Request) {
	tenant := h.tenant(r)
	renderAdmin(w, "login", TemplateData{Tenant: tenant})
}

func (h *AdminHandler) ProcessLogin(w http.ResponseWriter, r *http.Request) {
	tenant := h.tenant(r)
	email := strings.TrimSpace(r.FormValue("email"))
	password := r.FormValue("password")

	if email != tenant.AdminEmail || !services.CheckPassword(tenant.AdminPassword, password) {
		renderAdmin(w, "login", TemplateData{Tenant: tenant, Error: "Email atau password salah"})
		return
	}

	h.authSvc.SetAdminSession(w, r, tenant.ID)
	redirectf(w, r, "/"+tenant.Slug+"/admin")
}

func (h *AdminHandler) Logout(w http.ResponseWriter, r *http.Request) {
	tenant := h.tenant(r)
	h.authSvc.ClearAdminSession(w, r)
	redirectf(w, r, "/"+tenant.Slug+"/admin/login")
}

func (h *AdminHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	tenant := h.tenant(r)
	if _, ok := h.requireAuth(w, r); !ok {
		return
	}

	session, err := h.queueSvc.GetOrCreateSession(tenant.ID)
	if err != nil {
		renderAdmin(w, "dashboard", TemplateData{Tenant: tenant, Error: err.Error()})
		return
	}

	tickets, _ := h.queueSvc.GetTodayTickets(tenant.ID)

	type dashData struct {
		Session      *models.QueueSession
		Tickets      []*models.QueueTicket
		TotalWaiting int
		TotalDone    int
		TotalSkipped int
	}

	var waiting, done, skipped int
	for _, t := range tickets {
		switch t.Status {
		case "waiting":
			waiting++
		case "done":
			done++
		case "skipped":
			skipped++
		}
	}

	renderAdmin(w, "dashboard", TemplateData{
		Tenant: tenant,
		Data: dashData{
			Session:      session,
			Tickets:      tickets,
			TotalWaiting: waiting,
			TotalDone:    done,
			TotalSkipped: skipped,
		},
	})
}

func (h *AdminHandler) QueuePage(w http.ResponseWriter, r *http.Request) {
	tenant := h.tenant(r)
	if _, ok := h.requireAuth(w, r); !ok {
		return
	}

	session, err := h.queueSvc.GetOrCreateSession(tenant.ID)
	if err != nil {
		renderAdmin(w, "queue", TemplateData{Tenant: tenant, Error: err.Error()})
		return
	}

	tickets, _ := h.queueSvc.GetTodayTickets(tenant.ID)

	type queueData struct {
		Session *models.QueueSession
		Tickets []*models.QueueTicket
	}

	renderAdmin(w, "queue", TemplateData{
		Tenant: tenant,
		Data:   queueData{Session: session, Tickets: tickets},
	})
}

func (h *AdminHandler) OpenQueue(w http.ResponseWriter, r *http.Request) {
	tenant := h.tenant(r)
	if _, ok := h.requireAuth(w, r); !ok {
		return
	}
	h.queueSvc.OpenQueue(tenant.ID)
	redirectf(w, r, "/"+tenant.Slug+"/admin/queue")
}

func (h *AdminHandler) CloseQueue(w http.ResponseWriter, r *http.Request) {
	tenant := h.tenant(r)
	if _, ok := h.requireAuth(w, r); !ok {
		return
	}
	h.queueSvc.CloseQueue(tenant.ID)
	redirectf(w, r, "/"+tenant.Slug+"/admin/queue")
}

func (h *AdminHandler) NextQueue(w http.ResponseWriter, r *http.Request) {
	tenant := h.tenant(r)
	if _, ok := h.requireAuth(w, r); !ok {
		return
	}

	if _, err := h.queueSvc.CallNext(tenant); err != nil {
		session, _ := h.queueSvc.GetOrCreateSession(tenant.ID)
		tickets, _ := h.queueSvc.GetTodayTickets(tenant.ID)
		renderAdmin(w, "queue", TemplateData{
			Tenant: tenant,
			Error:  err.Error(),
			Data:   map[string]interface{}{"Session": session, "Tickets": tickets},
		})
		return
	}
	redirectf(w, r, "/"+tenant.Slug+"/admin/queue")
}

func (h *AdminHandler) CallNumber(w http.ResponseWriter, r *http.Request) {
	tenant := h.tenant(r)
	if _, ok := h.requireAuth(w, r); !ok {
		return
	}

	num, _ := strconv.Atoi(mux.Vars(r)["num"])
	if _, err := h.queueSvc.CallNumber(tenant, num); err != nil {
		redirectf(w, r, "/"+tenant.Slug+"/admin/queue?error="+err.Error())
		return
	}
	redirectf(w, r, "/"+tenant.Slug+"/admin/queue")
}

func (h *AdminHandler) SkipNumber(w http.ResponseWriter, r *http.Request) {
	tenant := h.tenant(r)
	if _, ok := h.requireAuth(w, r); !ok {
		return
	}

	num, _ := strconv.Atoi(mux.Vars(r)["num"])
	h.queueSvc.SkipNumber(tenant, num)
	redirectf(w, r, "/"+tenant.Slug+"/admin/queue")
}

func (h *AdminHandler) DoneNumber(w http.ResponseWriter, r *http.Request) {
	tenant := h.tenant(r)
	if _, ok := h.requireAuth(w, r); !ok {
		return
	}

	num, _ := strconv.Atoi(mux.Vars(r)["num"])
	h.queueSvc.DoneNumber(tenant, num)
	redirectf(w, r, "/"+tenant.Slug+"/admin/queue")
}

func (h *AdminHandler) SendNotification(w http.ResponseWriter, r *http.Request) {
	tenant := h.tenant(r)
	if _, ok := h.requireAuth(w, r); !ok {
		return
	}

	ticketID, _ := strconv.Atoi(mux.Vars(r)["ticketID"])
	message := strings.TrimSpace(r.FormValue("message"))

	ticket, err := h.queueSvc.GetTicketByID(ticketID)
	if err != nil || ticket == nil {
		redirectf(w, r, "/"+tenant.Slug+"/admin/queue")
		return
	}

	if err := h.notifSvc.SendCustom(tenant, ticket, message); err != nil {
		redirectf(w, r, fmt.Sprintf("/%s/admin/queue?error=%s", tenant.Slug, err.Error()))
		return
	}
	redirectf(w, r, "/"+tenant.Slug+"/admin/queue?flash=Notifikasi+berhasil+dikirim")
}

func (h *AdminHandler) ShowSettings(w http.ResponseWriter, r *http.Request) {
	tenant := h.tenant(r)
	if _, ok := h.requireAuth(w, r); !ok {
		return
	}
	renderAdmin(w, "settings", TemplateData{Tenant: tenant, Flash: r.URL.Query().Get("flash")})
}

func (h *AdminHandler) SaveSettings(w http.ResponseWriter, r *http.Request) {
	tenant := h.tenant(r)
	if _, ok := h.requireAuth(w, r); !ok {
		return
	}

	newSlug := strings.TrimSpace(r.FormValue("slug"))
	if newSlug == "" {
		newSlug = tenant.Slug
	}

	exists, _ := h.tenantRepo.SlugExists(newSlug, tenant.ID)
	if exists {
		renderAdmin(w, "settings", TemplateData{Tenant: tenant, Error: "URL sudah digunakan tenant lain"})
		return
	}

	estMin, _ := strconv.Atoi(r.FormValue("estimated_minutes"))
	if estMin <= 0 {
		estMin = tenant.EstimatedMinutes
	}
	notifyBefore, _ := strconv.Atoi(r.FormValue("notify_before"))
	if notifyBefore <= 0 {
		notifyBefore = tenant.NotifyBefore
	}

	tenant.Name = strings.TrimSpace(r.FormValue("name"))
	tenant.Slug = newSlug
	tenant.FonnteToken = strings.TrimSpace(r.FormValue("fonnte_token"))
	tenant.OpenTime = r.FormValue("open_time")
	tenant.CloseTime = r.FormValue("close_time")
	tenant.EstimatedMinutes = estMin
	tenant.NotifyBefore = notifyBefore
	tenant.WelcomeMessage = strings.TrimSpace(r.FormValue("welcome_message"))

	if newPwd := r.FormValue("new_password"); newPwd != "" {
		hashed, err := services.HashPassword(newPwd)
		if err == nil {
			h.tenantRepo.UpdatePassword(tenant.ID, hashed)
		}
	}

	if err := h.tenantRepo.Update(tenant); err != nil {
		renderAdmin(w, "settings", TemplateData{Tenant: tenant, Error: "Gagal menyimpan: " + err.Error()})
		return
	}

	redirectf(w, r, "/"+tenant.Slug+"/admin/settings?flash=Pengaturan+berhasil+disimpan")
}

func (h *AdminHandler) APIQueue(w http.ResponseWriter, r *http.Request) {
	tenant := h.tenant(r)
	if _, ok := h.authSvc.GetAdminSession(r); !ok {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	session, err := h.queueSvc.GetOrCreateSession(tenant.ID)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tickets, _ := h.queueSvc.GetTodayTickets(tenant.ID)
	jsonOK(w, map[string]interface{}{
		"session": session,
		"tickets": tickets,
	})
}
