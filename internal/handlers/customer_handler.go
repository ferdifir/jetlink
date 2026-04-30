package handlers

import (
	"net/http"
	"strings"

	"github.com/ferdifir/jetlink/internal/middleware"
	"github.com/ferdifir/jetlink/internal/models"
	"github.com/ferdifir/jetlink/internal/services"
)

type CustomerHandler struct {
	authSvc  *services.AuthService
	queueSvc *services.QueueService
}

func NewCustomerHandler(authSvc *services.AuthService, queueSvc *services.QueueService) *CustomerHandler {
	return &CustomerHandler{authSvc: authSvc, queueSvc: queueSvc}
}

func (h *CustomerHandler) tenant(r *http.Request) *models.Tenant {
	return r.Context().Value(middleware.TenantKey).(*models.Tenant)
}

func (h *CustomerHandler) Index(w http.ResponseWriter, r *http.Request) {
	tenant := h.tenant(r)
	_, customerID, _, ok := h.authSvc.GetCustomerSession(r)
	if !ok {
		redirectf(w, r, "/"+tenant.Slug+"/login")
		return
	}

	status, err := h.queueSvc.GetCustomerStatus(tenant, customerID)
	if err != nil {
		renderCustomer(w, "queue", TemplateData{Tenant: tenant, Error: err.Error()})
		return
	}

	renderCustomer(w, "queue", TemplateData{Tenant: tenant, Data: status})
}

func (h *CustomerHandler) ShowLogin(w http.ResponseWriter, r *http.Request) {
	tenant := h.tenant(r)
	renderCustomer(w, "login", TemplateData{Tenant: tenant})
}

func (h *CustomerHandler) ProcessLogin(w http.ResponseWriter, r *http.Request) {
	tenant := h.tenant(r)
	phone := strings.TrimSpace(r.FormValue("phone"))

	if phone == "" {
		renderCustomer(w, "login", TemplateData{Tenant: tenant, Error: "Nomor WhatsApp tidak boleh kosong"})
		return
	}

	phone = normalizePhone(phone)

	if err := h.authSvc.SendOTP(tenant, phone); err != nil {
		renderCustomer(w, "login", TemplateData{Tenant: tenant, Error: "Gagal mengirim OTP: " + err.Error()})
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "jetlink_otp_phone",
		Value:    phone,
		Path:     "/" + tenant.Slug,
		MaxAge:   300,
		HttpOnly: true,
	})

	redirectf(w, r, "/"+tenant.Slug+"/verify")
}

func (h *CustomerHandler) ShowVerify(w http.ResponseWriter, r *http.Request) {
	tenant := h.tenant(r)
	cookie, err := r.Cookie("jetlink_otp_phone")
	if err != nil || cookie.Value == "" {
		redirectf(w, r, "/"+tenant.Slug+"/login")
		return
	}
	renderCustomer(w, "verify", TemplateData{Tenant: tenant, Data: map[string]string{"phone": maskPhone(cookie.Value)}})
}

func (h *CustomerHandler) ProcessVerify(w http.ResponseWriter, r *http.Request) {
	tenant := h.tenant(r)

	cookie, err := r.Cookie("jetlink_otp_phone")
	if err != nil || cookie.Value == "" {
		redirectf(w, r, "/"+tenant.Slug+"/login")
		return
	}
	phone := cookie.Value
	code := strings.TrimSpace(r.FormValue("code"))

	valid, err := h.authSvc.VerifyOTP(tenant.ID, phone, code)
	if err != nil {
		renderCustomer(w, "verify", TemplateData{Tenant: tenant, Error: "Terjadi kesalahan, coba lagi"})
		return
	}
	if !valid {
		renderCustomer(w, "verify", TemplateData{
			Tenant: tenant,
			Error:  "Kode OTP salah atau sudah kadaluarsa",
			Data:   map[string]string{"phone": maskPhone(phone)},
		})
		return
	}

	customer, err := h.authSvc.GetOrCreateCustomer(tenant.ID, phone)
	if err != nil {
		renderCustomer(w, "verify", TemplateData{Tenant: tenant, Error: "Terjadi kesalahan, coba lagi"})
		return
	}

	http.SetCookie(w, &http.Cookie{Name: "jetlink_otp_phone", Path: "/" + tenant.Slug, MaxAge: -1})
	h.authSvc.SetCustomerSession(w, r, tenant.ID, customer.ID, phone)
	redirectf(w, r, "/"+tenant.Slug)
}

func (h *CustomerHandler) TakeQueue(w http.ResponseWriter, r *http.Request) {
	tenant := h.tenant(r)
	_, customerID, _, ok := h.authSvc.GetCustomerSession(r)
	if !ok {
		redirectf(w, r, "/"+tenant.Slug+"/login")
		return
	}

	ticket, err := h.queueSvc.TakeNumber(tenant, customerID)
	if err != nil {
		status, _ := h.queueSvc.GetCustomerStatus(tenant, customerID)
		renderCustomer(w, "queue", TemplateData{Tenant: tenant, Error: err.Error(), Data: status})
		return
	}

	_ = ticket
	redirectf(w, r, "/"+tenant.Slug)
}

func (h *CustomerHandler) APIStatus(w http.ResponseWriter, r *http.Request) {
	tenant := h.tenant(r)
	_, customerID, _, ok := h.authSvc.GetCustomerSession(r)
	if !ok {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	status, err := h.queueSvc.GetCustomerStatus(tenant, customerID)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, status)
}

func (h *CustomerHandler) Logout(w http.ResponseWriter, r *http.Request) {
	tenant := h.tenant(r)
	h.authSvc.ClearCustomerSession(w, r)
	redirectf(w, r, "/"+tenant.Slug+"/login")
}

func normalizePhone(phone string) string {
	phone = strings.ReplaceAll(phone, " ", "")
	phone = strings.ReplaceAll(phone, "-", "")
	if strings.HasPrefix(phone, "0") {
		phone = "62" + phone[1:]
	}
	if strings.HasPrefix(phone, "+") {
		phone = phone[1:]
	}
	return phone
}

func maskPhone(phone string) string {
	if len(phone) <= 6 {
		return phone
	}
	return phone[:4] + strings.Repeat("*", len(phone)-6) + phone[len(phone)-2:]
}
