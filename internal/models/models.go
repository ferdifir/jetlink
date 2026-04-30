package models

import "time"

type Tenant struct {
	ID               int
	Name             string
	Slug             string
	AdminEmail       string
	AdminPassword    string
	FonnteToken      string
	OpenTime         string
	CloseTime        string
	EstimatedMinutes int
	NotifyBefore     int
	IsActive         bool
	WelcomeMessage   string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type Customer struct {
	ID        int
	TenantID  int
	Phone     string
	Name      string
	CreatedAt time.Time
}

type OTPCode struct {
	ID        int
	TenantID  int
	Phone     string
	Code      string
	ExpiresAt time.Time
	Used      bool
	CreatedAt time.Time
}

type QueueSession struct {
	ID            int
	TenantID      int
	Date          time.Time
	CurrentNumber int
	NextNumber    int
	IsOpen        bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type QueueTicket struct {
	ID                  int
	TenantID            int
	QueueSessionID      int
	CustomerID          int
	Number              int
	Status              string
	NotifiedApproaching bool
	NotifiedCalled      bool
	Notes               string
	CreatedAt           time.Time
	UpdatedAt           time.Time
	CustomerPhone       string
	CustomerName        string
}

type QueueStatus struct {
	Ticket        *QueueTicket
	Session       *QueueSession
	Position      int
	EstimatedWait int
	TotalWaiting  int
}

type Superadmin struct {
	ID        int
	Email     string
	Password  string
	CreatedAt time.Time
}
