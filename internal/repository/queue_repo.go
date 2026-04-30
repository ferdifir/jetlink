package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/ferdifir/jetlink/internal/models"
)

type QueueRepo struct {
	db *sql.DB
}

func NewQueueRepo(db *sql.DB) *QueueRepo {
	return &QueueRepo{db: db}
}

func (r *QueueRepo) GetOrCreateSession(tenantID int) (*models.QueueSession, error) {
	return r.GetOrCreateSessionWithTenant(tenantID, nil)
}

func (r *QueueRepo) GetOrCreateSessionWithTenant(tenantID int, tenant *models.Tenant) (*models.QueueSession, error) {
	today := time.Now().Format("2006-01-02")
	s := &models.QueueSession{}

	err := r.db.QueryRow(
		`SELECT id, tenant_id, date, current_number, next_number, is_open, created_at, updated_at
		 FROM queue_sessions WHERE tenant_id=? AND date=?`,
		tenantID, today,
	).Scan(&s.ID, &s.TenantID, &s.Date, &s.CurrentNumber, &s.NextNumber, &s.IsOpen, &s.CreatedAt, &s.UpdatedAt)

	if err == sql.ErrNoRows {
		result, err := r.db.Exec(
			`INSERT INTO queue_sessions (tenant_id, date) VALUES (?, ?)`,
			tenantID, today,
		)
		if err != nil {
			return nil, fmt.Errorf("create queue session: %w", err)
		}
		id, _ := result.LastInsertId()
		return r.GetSessionByID(int(id))
	}
	if err != nil {
		return nil, fmt.Errorf("get queue session: %w", err)
	}

	if tenant != nil && !s.IsOpen && s.CurrentNumber == 0 {
		// Use Asia/Jakarta timezone as default for queue opening logic
		loc, err := time.LoadLocation("Asia/Jakarta")
		if err != nil {
			// Fallback to local timezone if Asia/Jakarta is not available
			loc = time.Now().Location()
		}
		now := time.Now().In(loc)
		
		openTime, err := time.ParseInLocation("15:04", tenant.OpenTime, loc)
		if err == nil && !now.Before(openTime) {
			closeTime, err := time.ParseInLocation("15:04", tenant.CloseTime, loc)
			if err == nil && now.Before(closeTime) {
				_, err = r.db.Exec(`UPDATE queue_sessions SET is_open=TRUE, updated_at=NOW() WHERE id=?`, s.ID)
				if err == nil {
					s.IsOpen = true
				}
			}
		}
	}

	return s, nil
}

func (r *QueueRepo) GetSessionByID(id int) (*models.QueueSession, error) {
	s := &models.QueueSession{}
	err := r.db.QueryRow(
		`SELECT id, tenant_id, date, current_number, next_number, is_open, created_at, updated_at
		 FROM queue_sessions WHERE id=?`, id,
	).Scan(&s.ID, &s.TenantID, &s.Date, &s.CurrentNumber, &s.NextNumber, &s.IsOpen, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (r *QueueRepo) OpenQueue(sessionID int) error {
	_, err := r.db.Exec(`UPDATE queue_sessions SET is_open=TRUE, updated_at=NOW() WHERE id=?`, sessionID)
	return err
}

func (r *QueueRepo) CloseQueue(sessionID int) error {
	_, err := r.db.Exec(`UPDATE queue_sessions SET is_open=FALSE, updated_at=NOW() WHERE id=?`, sessionID)
	return err
}

func (r *QueueRepo) SetCurrentNumber(sessionID, number int) error {
	_, err := r.db.Exec(
		`UPDATE queue_sessions SET current_number=?, updated_at=NOW() WHERE id=?`,
		number, sessionID,
	)
	return err
}

func (r *QueueRepo) IssueTicket(tenantID, sessionID, customerID, number int) (*models.QueueTicket, error) {
	result, err := r.db.Exec(
		`INSERT INTO queue_tickets (tenant_id, queue_session_id, customer_id, number) VALUES (?, ?, ?, ?)`,
		tenantID, sessionID, customerID, number,
	)
	if err != nil {
		return nil, fmt.Errorf("issue ticket: %w", err)
	}
	id, _ := result.LastInsertId()

	t := &models.QueueTicket{}
	err = r.db.QueryRow(
		`SELECT id, tenant_id, queue_session_id, customer_id, number, status,
		 notified_approaching, notified_called, COALESCE(notes,''), created_at, updated_at
		 FROM queue_tickets WHERE id=?`, id,
	).Scan(&t.ID, &t.TenantID, &t.QueueSessionID, &t.CustomerID, &t.Number, &t.Status,
		&t.NotifiedApproaching, &t.NotifiedCalled, &t.Notes, &t.CreatedAt, &t.UpdatedAt)
	return t, err
}

func (r *QueueRepo) IncrementNextNumber(sessionID int) (int, error) {
	var next int
	tx, err := r.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	if err := tx.QueryRow(
		`SELECT next_number FROM queue_sessions WHERE id=? FOR UPDATE`, sessionID,
	).Scan(&next); err != nil {
		return 0, err
	}

	if _, err := tx.Exec(
		`UPDATE queue_sessions SET next_number=next_number+1, updated_at=NOW() WHERE id=?`, sessionID,
	); err != nil {
		return 0, err
	}

	return next, tx.Commit()
}

func (r *QueueRepo) GetCustomerTicketToday(tenantID, customerID int) (*models.QueueTicket, error) {
	today := time.Now().Format("2006-01-02")
	t := &models.QueueTicket{}
	err := r.db.QueryRow(
		`SELECT qt.id, qt.tenant_id, qt.queue_session_id, qt.customer_id, qt.number, qt.status,
		 qt.notified_approaching, qt.notified_called, COALESCE(qt.notes,''), qt.created_at, qt.updated_at
		 FROM queue_tickets qt
		 JOIN queue_sessions qs ON qt.queue_session_id = qs.id
		 WHERE qt.tenant_id=? AND qt.customer_id=? AND qs.date=?`,
		tenantID, customerID, today,
	).Scan(&t.ID, &t.TenantID, &t.QueueSessionID, &t.CustomerID, &t.Number, &t.Status,
		&t.NotifiedApproaching, &t.NotifiedCalled, &t.Notes, &t.CreatedAt, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get customer ticket: %w", err)
	}
	return t, nil
}

func (r *QueueRepo) GetTodayTickets(tenantID int) ([]*models.QueueTicket, error) {
	today := time.Now().Format("2006-01-02")
	rows, err := r.db.Query(
		`SELECT qt.id, qt.tenant_id, qt.queue_session_id, qt.customer_id, qt.number, qt.status,
		 qt.notified_approaching, qt.notified_called, COALESCE(qt.notes,''), qt.created_at, qt.updated_at,
		 c.phone, COALESCE(c.name,'')
		 FROM queue_tickets qt
		 JOIN queue_sessions qs ON qt.queue_session_id = qs.id
		 JOIN customers c ON qt.customer_id = c.id
		 WHERE qt.tenant_id=? AND qs.date=?
		 ORDER BY qt.number ASC`,
		tenantID, today,
	)
	if err != nil {
		return nil, fmt.Errorf("get today tickets: %w", err)
	}
	defer rows.Close()

	var tickets []*models.QueueTicket
	for rows.Next() {
		t := &models.QueueTicket{}
		if err := rows.Scan(
			&t.ID, &t.TenantID, &t.QueueSessionID, &t.CustomerID, &t.Number, &t.Status,
			&t.NotifiedApproaching, &t.NotifiedCalled, &t.Notes, &t.CreatedAt, &t.UpdatedAt,
			&t.CustomerPhone, &t.CustomerName,
		); err != nil {
			return nil, err
		}
		tickets = append(tickets, t)
	}
	return tickets, nil
}

func (r *QueueRepo) GetWaitingTickets(sessionID int) ([]*models.QueueTicket, error) {
	rows, err := r.db.Query(
		`SELECT qt.id, qt.tenant_id, qt.queue_session_id, qt.customer_id, qt.number, qt.status,
		 qt.notified_approaching, qt.notified_called, COALESCE(qt.notes,''), qt.created_at, qt.updated_at,
		 c.phone, COALESCE(c.name,'')
		 FROM queue_tickets qt
		 JOIN customers c ON qt.customer_id = c.id
		 WHERE qt.queue_session_id=? AND qt.status='waiting'
		 ORDER BY qt.number ASC`,
		sessionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tickets []*models.QueueTicket
	for rows.Next() {
		t := &models.QueueTicket{}
		if err := rows.Scan(
			&t.ID, &t.TenantID, &t.QueueSessionID, &t.CustomerID, &t.Number, &t.Status,
			&t.NotifiedApproaching, &t.NotifiedCalled, &t.Notes, &t.CreatedAt, &t.UpdatedAt,
			&t.CustomerPhone, &t.CustomerName,
		); err != nil {
			return nil, err
		}
		tickets = append(tickets, t)
	}
	return tickets, nil
}

func (r *QueueRepo) UpdateTicketStatus(ticketID int, status string) error {
	_, err := r.db.Exec(
		`UPDATE queue_tickets SET status=?, updated_at=NOW() WHERE id=?`, status, ticketID,
	)
	return err
}

func (r *QueueRepo) GetTicketByNumber(sessionID, number int) (*models.QueueTicket, error) {
	t := &models.QueueTicket{}
	err := r.db.QueryRow(
		`SELECT qt.id, qt.tenant_id, qt.queue_session_id, qt.customer_id, qt.number, qt.status,
		 qt.notified_approaching, qt.notified_called, COALESCE(qt.notes,''), qt.created_at, qt.updated_at,
		 c.phone, COALESCE(c.name,'')
		 FROM queue_tickets qt
		 JOIN customers c ON qt.customer_id = c.id
		 WHERE qt.queue_session_id=? AND qt.number=?`,
		sessionID, number,
	).Scan(
		&t.ID, &t.TenantID, &t.QueueSessionID, &t.CustomerID, &t.Number, &t.Status,
		&t.NotifiedApproaching, &t.NotifiedCalled, &t.Notes, &t.CreatedAt, &t.UpdatedAt,
		&t.CustomerPhone, &t.CustomerName,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return t, err
}

func (r *QueueRepo) GetTicketByID(id int) (*models.QueueTicket, error) {
	t := &models.QueueTicket{}
	err := r.db.QueryRow(
		`SELECT qt.id, qt.tenant_id, qt.queue_session_id, qt.customer_id, qt.number, qt.status,
		 qt.notified_approaching, qt.notified_called, COALESCE(qt.notes,''), qt.created_at, qt.updated_at,
		 c.phone, COALESCE(c.name,'')
		 FROM queue_tickets qt
		 JOIN customers c ON qt.customer_id = c.id
		 WHERE qt.id=?`, id,
	).Scan(
		&t.ID, &t.TenantID, &t.QueueSessionID, &t.CustomerID, &t.Number, &t.Status,
		&t.NotifiedApproaching, &t.NotifiedCalled, &t.Notes, &t.CreatedAt, &t.UpdatedAt,
		&t.CustomerPhone, &t.CustomerName,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return t, err
}

func (r *QueueRepo) MarkNotifiedApproaching(ticketID int) error {
	_, err := r.db.Exec(
		`UPDATE queue_tickets SET notified_approaching=TRUE, updated_at=NOW() WHERE id=?`, ticketID,
	)
	return err
}

func (r *QueueRepo) MarkNotifiedCalled(ticketID int) error {
	_, err := r.db.Exec(
		`UPDATE queue_tickets SET notified_called=TRUE, updated_at=NOW() WHERE id=?`, ticketID,
	)
	return err
}

func (r *QueueRepo) CountByStatus(sessionID int, status string) (int, error) {
	var count int
	err := r.db.QueryRow(
		`SELECT COUNT(*) FROM queue_tickets WHERE queue_session_id=? AND status=?`,
		sessionID, status,
	).Scan(&count)
	return count, err
}
