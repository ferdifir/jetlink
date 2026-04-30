package services

import (
	"crypto/sha256"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/ferdifir/jetlink/internal/models"
	"github.com/ferdifir/jetlink/internal/repository"
)

type AuthService struct {
	customerRepo *repository.CustomerRepo
	fonnte       *FonnteClient
}

func NewAuthService(customerRepo *repository.CustomerRepo, fonnte *FonnteClient) *AuthService {
	return &AuthService{
		customerRepo: customerRepo,
		fonnte:       fonnte,
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
	expiry := time.Now().Add(7 * 24 * time.Hour)
	cookieValue := fmt.Sprintf("%d:%d:%s:%d", tenantID, customerID, phone, expiry.Unix())
	http.SetCookie(w, &http.Cookie{
		Name:     "jetlink_customer",
		Value:    signCookie(cookieValue),
		Path:     "/",
		Expires:  expiry,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	return nil
}

func (s *AuthService) GetCustomerSession(r *http.Request) (tenantID, customerID int, phone string, ok bool) {
	cookie, err := r.Cookie("jetlink_customer")
	if err != nil {
		return
	}
	parts, valid := verifyCookie(cookie.Value)
	if !valid || len(parts) < 4 {
		return
	}
	expiry, err := strconv.ParseInt(parts[3], 10, 64)
	if err != nil || time.Now().Unix() > expiry {
		return
	}
	tenantID, _ = strconv.Atoi(parts[0])
	customerID, _ = strconv.Atoi(parts[1])
	phone = parts[2]
	ok = true
	return
}

func (s *AuthService) ClearCustomerSession(w http.ResponseWriter, r *http.Request) error {
	http.SetCookie(w, &http.Cookie{
		Name:     "jetlink_customer",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
	return nil
}

func (s *AuthService) SetAdminSession(w http.ResponseWriter, r *http.Request, tenantID int) error {
	expiry := time.Now().Add(7 * 24 * time.Hour)
	cookieValue := fmt.Sprintf("%d:%d", tenantID, expiry.Unix())
	http.SetCookie(w, &http.Cookie{
		Name:     "jetlink_admin",
		Value:    signCookie(cookieValue),
		Path:     "/",
		Expires:  expiry,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	return nil
}

func (s *AuthService) GetAdminSession(r *http.Request) (tenantID int, ok bool) {
	cookie, err := r.Cookie("jetlink_admin")
	if err != nil {
		return
	}
	parts, valid := verifyCookie(cookie.Value)
	if !valid || len(parts) < 2 {
		return
	}
	expiry, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || time.Now().Unix() > expiry {
		return
	}
	tenantID, _ = strconv.Atoi(parts[0])
	ok = true
	return
}

func (s *AuthService) ClearAdminSession(w http.ResponseWriter, r *http.Request) error {
	http.SetCookie(w, &http.Cookie{
		Name:     "jetlink_admin",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
	return nil
}

func (s *AuthService) SetSuperadminSession(w http.ResponseWriter, r *http.Request, id int) error {
	expiry := time.Now().Add(7 * 24 * time.Hour)
	cookieValue := fmt.Sprintf("%d:%d", id, expiry.Unix())
	http.SetCookie(w, &http.Cookie{
		Name:     "jetlink_superadmin",
		Value:    signCookie(cookieValue),
		Path:     "/",
		Expires:  expiry,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	return nil
}

func (s *AuthService) GetSuperadminSession(r *http.Request) (id int, ok bool) {
	cookie, err := r.Cookie("jetlink_superadmin")
	if err != nil {
		return
	}
	parts, valid := verifyCookie(cookie.Value)
	if !valid || len(parts) < 2 {
		return
	}
	expiry, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || time.Now().Unix() > expiry {
		return
	}
	id, _ = strconv.Atoi(parts[0])
	ok = true
	return
}

func (s *AuthService) ClearSuperadminSession(w http.ResponseWriter, r *http.Request) error {
	http.SetCookie(w, &http.Cookie{
		Name:     "jetlink_superadmin",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
	return nil
}

// Cookie signing using simple HMAC-like approach with session secret
var cookieSecret = []byte("jetlink-secret-change-in-prod-2025")

func signCookie(value string) string {
	h := sha256.Sum256(append([]byte(value), cookieSecret...))
	return fmt.Sprintf("%s.%x", value, h[:8])
}

func verifyCookie(signed string) ([]string, bool) {
	parts := strings.SplitN(signed, ".", 2)
	if len(parts) != 2 {
		return nil, false
	}
	expected := signCookie(parts[0])
	return strings.Split(parts[0], ":"), signed == expected
}

func HashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(b), err
}

func CheckPassword(hashed, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(plain)) == nil
}
