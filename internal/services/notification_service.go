package services

import (
	"fmt"
	"log"

	"github.com/ferdifir/jetlink/internal/models"
	"github.com/ferdifir/jetlink/internal/repository"
)

type NotificationService struct {
	queueRepo *repository.QueueRepo
	fonnte    *FonnteClient
}

func NewNotificationService(queueRepo *repository.QueueRepo, fonnte *FonnteClient) *NotificationService {
	return &NotificationService{queueRepo: queueRepo, fonnte: fonnte}
}

func (s *NotificationService) NotifyAffected(tenant *models.Tenant, sessionID, currentNumber int) {
	waiting, err := s.queueRepo.GetWaitingTickets(sessionID)
	if err != nil {
		log.Printf("[notify] error getting waiting tickets: %v", err)
		return
	}

	for _, ticket := range waiting {
		position := ticket.Number - currentNumber
		if position <= 0 {
			continue
		}

		if position <= tenant.NotifyBefore && !ticket.NotifiedApproaching {
			msg := fmt.Sprintf(
				"⏰ *Antrian Anda Semakin Dekat!*\n\nHalo! Nomor antrian Anda di *%s* sudah dekat.\n\n"+
					"📋 Nomor Anda: *%d*\n"+
					"🔢 Sedang dilayani: *%d*\n"+
					"👥 Sisa antrian sebelum Anda: *%d*\n"+
					"⏱️ Estimasi tunggu: *~%d menit*\n\n"+
					"Harap bersiap di lokasi.",
				tenant.Name, ticket.Number, currentNumber, position-1, position*tenant.EstimatedMinutes,
			)
			if err := s.fonnte.SendMessage(ticket.CustomerPhone, msg); err != nil {
				log.Printf("[notify] error sending approaching to %s: %v", ticket.CustomerPhone, err)
			} else {
				s.queueRepo.MarkNotifiedApproaching(ticket.ID)
			}
		}
	}

	called, err := s.queueRepo.GetTicketByNumber(sessionID, currentNumber)
	if err != nil || called == nil {
		return
	}

	if !called.NotifiedCalled {
		msg := fmt.Sprintf(
			"🔔 *Giliran Anda Tiba!*\n\nNomor antrian *%d* di *%s* sekarang dipanggil.\n\n"+
				"Silakan segera menuju loket/tempat pelayanan.\n\n"+
				"Terima kasih telah menunggu! 🙏",
			called.Number, tenant.Name,
		)
		if err := s.fonnte.SendMessage(called.CustomerPhone, msg); err != nil {
			log.Printf("[notify] error sending called to %s: %v", called.CustomerPhone, err)
		} else {
			s.queueRepo.MarkNotifiedCalled(called.ID)
		}
	}
}

func (s *NotificationService) SendCustom(tenant *models.Tenant, ticket *models.QueueTicket, message string) error {
	if message == "" {
		return fmt.Errorf("pesan tidak boleh kosong")
	}
	return s.fonnte.SendMessage(ticket.CustomerPhone, message)
}
