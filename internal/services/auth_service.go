package services

import (
	"fmt"
	"math/rand"
	"net/http"

	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"

	"github.com/ferdifir/jetlink/internal/models"
	"github.com/ferdifir/jetlink/internal/repository"
)

const (
	sessionCustomer   = "jetlink_customer"
	sessionAdmin      = "jetlink_admin"
	sessionSuperadmin = "jetlink_superadmin"
)

type AuthService struct {
	customerRepo *repository.CustomerRepo
	fonnte       *FonnteClient
	store        *sessions.CookieStore
}

func NewAuthService(customerRepo *repository.CustomerRepo, fonnte *FonnteClient, store *sessions.CookieStore) *AuthService {
	return &AuthService{
		customerRepo: customerRepo,
		fonnte:       fonnte,
		store:        store,
	}
}

func (s *AuthService) SendOTP(tenant *models.Tenant, phone string) error {
	code := fmt.Sprintf("%06d", rand.Intn(1000000))

	if err := s.customerRepo.SaveOTP(tenant.ID, phone, code); err != nil {
		return fmt.Errorf("save otp: %w", err)
	}

	message := fmt.Sprintf("Kode OTP Anda untuk %s adalah: *%s*\nBerlaku 5 menit. Jangan bagikan ke siapapun.",
		tenant.Name, code)

	return s.fonnte.SendMessage(phone, message)
}

func (s *AuthService) VerifyOTP(tenantID int, phone, code string) (bool, error) {
	return s.customerRepo.VerifyOTP(tenantID, phone, code)
}

func (s *AuthService) GetOrCreateCustomer(tenantID int, phone string) (*models.Customer, error) {
	c, err := s.customerRepo.GetByPhone(tenantID, phone)
	if err != nil {
		return nil, err
	}
	if c != nil {
		return c, nil
	}

	c = &models.Customer{
		TenantID: tenantID,
		Phone:    phone,
	}
	if err := s.customerRepo.Create(c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *AuthService) SetCustomerSession(w http.ResponseWriter, r *http.Request, tenantID, customerID int, phone string) error {
	sess, err := s.store.Get(r, sessionCustomer)
	if err != nil {
		sess, _ = s.store.New(r, sessionCustomer)
	}
	sess.Values["tenant_id"] = tenantID
	sess.Values["customer_id"] = customerID
	sess.Values["phone"] = phone
	return sess.Save(r, w)
}

func (s *AuthService) GetCustomerSession(r *http.Request) (tenantID, customerID int, phone string, ok bool) {
	sess, err := s.store.Get(r, sessionCustomer)
	if err != nil {
		return
	}
	tid, tok := sess.Values["tenant_id"].(int)
	cid, cok := sess.Values["customer_id"].(int)
	ph, phok := sess.Values["phone"].(string)
	if !tok || !cok || !phok {
		return
	}
	return tid, cid, ph, true
}

func (s *AuthService) ClearCustomerSession(w http.ResponseWriter, r *http.Request) error {
	sess, _ := s.store.Get(r, sessionCustomer)
	sess.Options.MaxAge = -1
	return sess.Save(r, w)
}

func (s *AuthService) SetAdminSession(w http.ResponseWriter, r *http.Request, tenantID int) error {
	sess, err := s.store.Get(r, sessionAdmin)
	if err != nil {
		sess, _ = s.store.New(r, sessionAdmin)
	}
	sess.Values["tenant_id"] = tenantID
	return sess.Save(r, w)
}

func (s *AuthService) GetAdminSession(r *http.Request) (tenantID int, ok bool) {
	sess, err := s.store.Get(r, sessionAdmin)
	if err != nil {
		return
	}
	tid, tok := sess.Values["tenant_id"].(int)
	return tid, tok
}

func (s *AuthService) ClearAdminSession(w http.ResponseWriter, r *http.Request) error {
	sess, _ := s.store.Get(r, sessionAdmin)
	sess.Options.MaxAge = -1
	return sess.Save(r, w)
}

func (s *AuthService) SetSuperadminSession(w http.ResponseWriter, r *http.Request, id int) error {
	sess, err := s.store.Get(r, sessionSuperadmin)
	if err != nil {
		sess, _ = s.store.New(r, sessionSuperadmin)
	}
	sess.Values["superadmin_id"] = id
	return sess.Save(r, w)
}

func (s *AuthService) GetSuperadminSession(r *http.Request) (id int, ok bool) {
	sess, err := s.store.Get(r, sessionSuperadmin)
	if err != nil {
		return
	}
	id, ok = sess.Values["superadmin_id"].(int)
	return
}

func (s *AuthService) ClearSuperadminSession(w http.ResponseWriter, r *http.Request) error {
	sess, _ := s.store.Get(r, sessionSuperadmin)
	sess.Options.MaxAge = -1
	return sess.Save(r, w)
}

func HashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(b), err
}

func CheckPassword(hashed, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(plain)) == nil
}
