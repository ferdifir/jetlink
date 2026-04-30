package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"

	"github.com/ferdifir/jetlink/internal/models"
	"github.com/ferdifir/jetlink/internal/repository"
	"github.com/ferdifir/jetlink/internal/services"
)

type SuperadminHandler struct {
	authSvc    *services.AuthService
	tenantRepo *repository.TenantRepo
}

func NewSuperadminHandler(authSvc *services.AuthService, tenantRepo *repository.TenantRepo) *SuperadminHandler {
	return &SuperadminHandler{authSvc: authSvc, tenantRepo: tenantRepo}
}

func (h *SuperadminHandler) requireAuth(w http.ResponseWriter, r *http.Request) (int, bool) {
	id, ok := h.authSvc.GetSuperadminSession(r)
	if !ok {
		redirectf(w, r, "/superadmin/login")
		return 0, false
	}
	return id, true
}

func (h *SuperadminHandler) ShowLogin(w http.ResponseWriter, r *http.Request) {
	renderSuperadmin(w, "login", TemplateData{})
}

func (h *SuperadminHandler) ProcessLogin(w http.ResponseWriter, r *http.Request) {
	email := strings.TrimSpace(r.FormValue("email"))
	password := r.FormValue("password")

	sa, err := h.tenantRepo.GetSuperadminByEmail(email)
	if err != nil || sa == nil || !services.CheckPassword(sa.Password, password) {
		renderSuperadmin(w, "login", TemplateData{Error: "Email atau password salah"})
		return
	}

	h.authSvc.SetSuperadminSession(w, r, sa.ID)
	redirectf(w, r, "/superadmin")
}

func (h *SuperadminHandler) Logout(w http.ResponseWriter, r *http.Request) {
	h.authSvc.ClearSuperadminSession(w, r)
	redirectf(w, r, "/superadmin/login")
}

func (h *SuperadminHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireAuth(w, r); !ok {
		return
	}

	tenants, err := h.tenantRepo.List()
	if err != nil {
		renderSuperadmin(w, "dashboard", TemplateData{Error: err.Error()})
		return
	}

	renderSuperadmin(w, "dashboard", TemplateData{
		Flash: r.URL.Query().Get("flash"),
		Error: r.URL.Query().Get("error"),
		Data:  tenants,
	})
}

func (h *SuperadminHandler) CreateTenant(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireAuth(w, r); !ok {
		return
	}

	slug := strings.ToLower(strings.TrimSpace(r.FormValue("slug")))
	name := strings.TrimSpace(r.FormValue("name"))
	email := strings.TrimSpace(r.FormValue("admin_email"))
	password := r.FormValue("admin_password")

	if slug == "" || name == "" || email == "" || password == "" {
		redirectf(w, r, "/superadmin?error=Semua+field+harus+diisi")
		return
	}

	exists, _ := h.tenantRepo.SlugExists(slug, 0)
	if exists {
		redirectf(w, r, "/superadmin?error=Slug+sudah+digunakan")
		return
	}

	hashed, err := services.HashPassword(password)
	if err != nil {
		redirectf(w, r, "/superadmin?error=Gagal+memproses+password")
		return
	}

	estMin, _ := strconv.Atoi(r.FormValue("estimated_minutes"))
	if estMin <= 0 {
		estMin = 5
	}

	tenant := &models.Tenant{
		Name:             name,
		Slug:             slug,
		AdminEmail:       email,
		AdminPassword:    hashed,
		OpenTime:         "08:00",
		CloseTime:        "17:00",
		EstimatedMinutes: estMin,
		NotifyBefore:     3,
		IsActive:         true,
	}

	if err := h.tenantRepo.Create(tenant); err != nil {
		redirectf(w, r, "/superadmin?error=Gagal+membuat+tenant:+"+err.Error())
		return
	}

	redirectf(w, r, "/superadmin?flash=Tenant+berhasil+dibuat")
}

func (h *SuperadminHandler) UpdateTenant(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireAuth(w, r); !ok {
		return
	}

	id, _ := strconv.Atoi(mux.Vars(r)["id"])
	tenant, err := h.tenantRepo.GetByID(id)
	if err != nil || tenant == nil {
		redirectf(w, r, "/superadmin?error=Tenant+tidak+ditemukan")
		return
	}

	tenant.Name = strings.TrimSpace(r.FormValue("name"))
	tenant.IsActive = r.FormValue("is_active") == "1"

	if err := h.tenantRepo.Update(tenant); err != nil {
		redirectf(w, r, "/superadmin?error="+err.Error())
		return
	}

	redirectf(w, r, "/superadmin?flash=Tenant+berhasil+diupdate")
}

func (h *SuperadminHandler) DeleteTenant(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireAuth(w, r); !ok {
		return
	}

	id, _ := strconv.Atoi(mux.Vars(r)["id"])
	if err := h.tenantRepo.Delete(id); err != nil {
		redirectf(w, r, "/superadmin?error="+err.Error())
		return
	}
	redirectf(w, r, "/superadmin?flash=Tenant+berhasil+dinonaktifkan")
}
