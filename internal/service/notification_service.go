package service

import (
	"fmt"
	"log"
	"net/smtp"
	"os"

	"github.com/dedehudianto12/simbu-tech-backend/internal/model"
)

// NotificationConfig holds SMTP settings read from environment variables.
type NotificationConfig struct {
	SMTPHost     string
	SMTPPort     string
	SMTPUser     string
	SMTPPassword string
	FromEmail    string
}

// NotificationService sends email notifications (fire-and-forget).
type NotificationService struct {
	cfg NotificationConfig
}

// NewNotificationService creates a NotificationService from env vars.
func NewNotificationService(cfg NotificationConfig) *NotificationService {
	if cfg.SMTPHost == "" {
		cfg.SMTPHost = os.Getenv("SMTP_HOST")
	}
	if cfg.SMTPPort == "" {
		cfg.SMTPPort = os.Getenv("SMTP_PORT")
	}
	if cfg.SMTPUser == "" {
		cfg.SMTPUser = os.Getenv("SMTP_USER")
	}
	if cfg.SMTPPassword == "" {
		cfg.SMTPPassword = os.Getenv("SMTP_PASSWORD")
	}
	if cfg.FromEmail == "" {
		cfg.FromEmail = os.Getenv("SMTP_FROM_EMAIL")
	}
	return &NotificationService{cfg: cfg}
}

func (s *NotificationService) sendEmail(to, subject, body string) {
	if s.cfg.SMTPHost == "" {
		log.Printf("notification: SMTP not configured, skipping email to %s", to)
		return
	}

	addr := fmt.Sprintf("%s:%s", s.cfg.SMTPHost, s.cfg.SMTPPort)
	msg := []byte(fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		s.cfg.FromEmail, to, subject, body))

	var auth smtp.Auth
	if s.cfg.SMTPUser != "" {
		auth = smtp.PlainAuth("", s.cfg.SMTPUser, s.cfg.SMTPPassword, s.cfg.SMTPHost)
	}

	if err := smtp.SendMail(addr, auth, s.cfg.FromEmail, []string{to}, msg); err != nil {
		log.Printf("notification: failed to send email to %s: %v", to, err)
	}
}

// SendTicketCreated notifies the requester that their ticket was received.
func (s *NotificationService) SendTicketCreated(ticket model.Ticket) {
	subject := fmt.Sprintf("Your support ticket has been received — %s", ticket.TicketNumber)
	slaInfo := "N/A"
	if ticket.SlaDueAt != nil {
		slaInfo = ticket.SlaDueAt.Format("2006-01-02 15:04:05 MST")
	}
	body := fmt.Sprintf(
		"Hello %s,\n\n"+
			"Your support ticket has been received.\n\n"+
			"Ticket Number: %s\n"+
			"Subject: %s\n"+
			"SLA Due: %s\n\n"+
			"You can check your ticket status at any time using your ticket number.\n\n"+
			"Best regards,\nPT Simbu Teknologi Indonesia",
		ticket.RequesterName, ticket.TicketNumber, ticket.Subject, slaInfo,
	)
	s.sendEmail(ticket.RequesterEmail, subject, body)
}

// SendStatusUpdated notifies the requester that their ticket status changed.
func (s *NotificationService) SendStatusUpdated(ticket model.Ticket) {
	subject := fmt.Sprintf("Your ticket %s has been updated", ticket.TicketNumber)
	body := fmt.Sprintf(
		"Hello %s,\n\n"+
			"Your ticket has been updated.\n\n"+
			"Ticket Number: %s\n"+
			"Subject: %s\n"+
			"New Status: %s\n\n"+
			"Best regards,\nPT Simbu Teknologi Indonesia",
		ticket.RequesterName, ticket.TicketNumber, ticket.Subject, ticket.Status,
	)
	s.sendEmail(ticket.RequesterEmail, subject, body)
}
