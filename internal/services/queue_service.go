package services

import (
	"fmt"

	"github.com/ferdifir/jetlink/internal/models"
	"github.com/ferdifir/jetlink/internal/repository"
)

type QueueService struct {
	queueRepo  *repository.QueueRepo
	tenantRepo *repository.TenantRepo
	notif      *NotificationService
}

func NewQueueService(queueRepo *repository.QueueRepo, tenantRepo *repository.TenantRepo) *QueueService {
	return &QueueService{queueRepo: queueRepo, tenantRepo: tenantRepo}
}

func (s *QueueService) SetNotificationService(notif *NotificationService) {
	s.notif = notif
}

func (s *QueueService) GetOrCreateSession(tenantID int) (*models.QueueSession, error) {
	return s.queueRepo.GetOrCreateSession(tenantID)
}

func (s *QueueService) GetOrCreateSessionWithTenant(tenant *models.Tenant) (*models.QueueSession, error) {
	return s.queueRepo.GetOrCreateSessionWithTenant(tenant.ID, tenant)
}

func (s *QueueService) TakeNumber(tenant *models.Tenant, customerID int) (*models.QueueTicket, error) {
	session, err := s.queueRepo.GetOrCreateSessionWithTenant(tenant.ID, tenant)
	if err != nil {
		return nil, err
	}
	if !session.IsOpen {
		return nil, fmt.Errorf("antrian sedang tutup")
	}

	existing, err := s.queueRepo.GetCustomerTicketToday(tenant.ID, customerID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf("Anda sudah mengambil nomor antrian hari ini (nomor %d)", existing.Number)
	}

	number, err := s.queueRepo.IncrementNextNumber(session.ID)
	if err != nil {
		return nil, fmt.Errorf("increment number: %w", err)
	}

	ticket, err := s.queueRepo.IssueTicket(tenant.ID, session.ID, customerID, number)
	if err != nil {
		return nil, fmt.Errorf("issue ticket: %w", err)
	}
	return ticket, nil
}

func (s *QueueService) GetCustomerStatus(tenant *models.Tenant, customerID int) (*models.QueueStatus, error) {
	session, err := s.queueRepo.GetOrCreateSessionWithTenant(tenant.ID, tenant)
	if err != nil {
		return nil, err
	}

	ticket, err := s.queueRepo.GetCustomerTicketToday(tenant.ID, customerID)
	if err != nil {
		return nil, err
	}

	totalWaiting, _ := s.queueRepo.CountByStatus(session.ID, "waiting")

	status := &models.QueueStatus{
		Ticket:       ticket,
		Session:      session,
		TotalWaiting: totalWaiting,
	}

	if ticket != nil && session.CurrentNumber < ticket.Number {
		status.Position = ticket.Number - session.CurrentNumber
		status.EstimatedWait = status.Position * tenant.EstimatedMinutes
	}

	return status, nil
}

func (s *QueueService) CallNext(tenant *models.Tenant) (*models.QueueTicket, error) {
	session, err := s.queueRepo.GetOrCreateSessionWithTenant(tenant.ID, tenant)
	if err != nil {
		return nil, err
	}

	waiting, err := s.queueRepo.GetWaitingTickets(session.ID)
	if err != nil {
		return nil, err
	}
	if len(waiting) == 0 {
		return nil, fmt.Errorf("tidak ada antrian yang menunggu")
	}

	next := waiting[0]
	if err := s.queueRepo.SetCurrentNumber(session.ID, next.Number); err != nil {
		return nil, err
	}
	if err := s.queueRepo.UpdateTicketStatus(next.ID, "called"); err != nil {
		return nil, err
	}

	if s.notif != nil {
		go s.notif.NotifyAffected(tenant, session.ID, next.Number)
	}

	return next, nil
}

func (s *QueueService) CallNumber(tenant *models.Tenant, number int) (*models.QueueTicket, error) {
	session, err := s.queueRepo.GetOrCreateSessionWithTenant(tenant.ID, tenant)
	if err != nil {
		return nil, err
	}

	ticket, err := s.queueRepo.GetTicketByNumber(session.ID, number)
	if err != nil {
		return nil, err
	}
	if ticket == nil {
		return nil, fmt.Errorf("nomor antrian %d tidak ditemukan", number)
	}

	if err := s.queueRepo.SetCurrentNumber(session.ID, number); err != nil {
		return nil, err
	}
	if err := s.queueRepo.UpdateTicketStatus(ticket.ID, "called"); err != nil {
		return nil, err
	}

	if s.notif != nil {
		go s.notif.NotifyAffected(tenant, session.ID, number)
	}

	return ticket, nil
}

func (s *QueueService) SkipNumber(tenant *models.Tenant, number int) error {
	session, err := s.queueRepo.GetOrCreateSessionWithTenant(tenant.ID, tenant)
	if err != nil {
		return err
	}

	ticket, err := s.queueRepo.GetTicketByNumber(session.ID, number)
	if err != nil {
		return err
	}
	if ticket == nil {
		return fmt.Errorf("nomor antrian %d tidak ditemukan", number)
	}

	return s.queueRepo.UpdateTicketStatus(ticket.ID, "skipped")
}

func (s *QueueService) DoneNumber(tenant *models.Tenant, number int) error {
	session, err := s.queueRepo.GetOrCreateSessionWithTenant(tenant.ID, tenant)
	if err != nil {
		return err
	}

	ticket, err := s.queueRepo.GetTicketByNumber(session.ID, number)
	if err != nil {
		return err
	}
	if ticket == nil {
		return fmt.Errorf("nomor antrian %d tidak ditemukan", number)
	}

	return s.queueRepo.UpdateTicketStatus(ticket.ID, "done")
}

func (s *QueueService) OpenQueue(tenantID int) error {
	session, err := s.queueRepo.GetOrCreateSession(tenantID)
	if err != nil {
		return err
	}
	return s.queueRepo.OpenQueue(session.ID)
}

func (s *QueueService) CloseQueue(tenantID int) error {
	session, err := s.queueRepo.GetOrCreateSession(tenantID)
	if err != nil {
		return err
	}
	return s.queueRepo.CloseQueue(session.ID)
}

func (s *QueueService) GetTodayTickets(tenantID int) ([]*models.QueueTicket, error) {
	return s.queueRepo.GetTodayTickets(tenantID)
}

func (s *QueueService) GetTicketByID(id int) (*models.QueueTicket, error) {
	return s.queueRepo.GetTicketByID(id)
}
