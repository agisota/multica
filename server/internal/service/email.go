package service

import (
	"fmt"
	"html"
	"os"
	"strings"

	"github.com/resend/resend-go/v2"
)

type EmailService struct {
	client    *resend.Client
	fromEmail string
	contactTo string
}

func NewEmailService() *EmailService {
	apiKey := os.Getenv("RESEND_API_KEY")
	fromAddress := strings.TrimSpace(os.Getenv("RESEND_FROM_EMAIL"))
	if fromAddress == "" {
		fromAddress = "hi@zed.md"
	}
	contactTo := strings.TrimSpace(os.Getenv("CONTACT_TO_EMAIL"))
	if contactTo == "" {
		contactTo = fromAddress
	}
	from := fromAddress
	// Wrap bare email in "Multica <email>" format for Resend
	if !strings.Contains(from, "<") {
		from = fmt.Sprintf("Multica <%s>", from)
	}

	var client *resend.Client
	if apiKey != "" {
		client = resend.NewClient(apiKey)
	}

	return &EmailService{
		client:    client,
		fromEmail: from,
		contactTo: contactTo,
	}
}

func (s *EmailService) SendVerificationCode(to, code string) error {
	if s.client == nil {
		fmt.Printf("[DEV] Verification code for %s: %s\n", to, code)
		return nil
	}

	params := &resend.SendEmailRequest{
		From:    s.fromEmail,
		To:      []string{to},
		Subject: "Your Multica verification code",
		Html: fmt.Sprintf(
			`<div style="font-family: sans-serif; max-width: 400px; margin: 0 auto;">
				<h2>Your verification code</h2>
				<p style="font-size: 32px; font-weight: bold; letter-spacing: 8px; margin: 24px 0;">%s</p>
				<p>This code expires in 10 minutes.</p>
				<p style="color: #666; font-size: 14px;">If you didn't request this code, you can safely ignore this email.</p>
			</div>`, code),
	}

	resp, err := s.client.Emails.Send(params)
	if err == nil && resp != nil {
		fmt.Printf("[EMAIL] verification_code_sent to=%s from=%s resend_id=%s\n", to, s.fromEmail, resp.Id)
	}
	return err
}

func (s *EmailService) SendLandingContactMessage(name, email, company, message string) error {
	if s.client == nil {
		fmt.Printf("[DEV] Landing contact from %s <%s> (%s): %s\n", name, email, company, message)
		return nil
	}

	var details []string
	details = append(details, fmt.Sprintf("<p><strong>Имя:</strong> %s</p>", html.EscapeString(name)))
	details = append(details, fmt.Sprintf("<p><strong>Email:</strong> %s</p>", html.EscapeString(email)))
	if strings.TrimSpace(company) != "" {
		details = append(details, fmt.Sprintf("<p><strong>Компания:</strong> %s</p>", html.EscapeString(company)))
	}
	details = append(details, fmt.Sprintf(
		"<p><strong>Сообщение:</strong></p><div style=\"white-space: pre-wrap; border: 1px solid #e5e7eb; border-radius: 8px; padding: 12px;\">%s</div>",
		html.EscapeString(message),
	))

	params := &resend.SendEmailRequest{
		From:    s.fromEmail,
		To:      []string{s.contactTo},
		ReplyTo: email,
		Subject: fmt.Sprintf("Новая заявка с лендинга Multica от %s", name),
		Html: fmt.Sprintf(
			`<div style="font-family: sans-serif; max-width: 640px; margin: 0 auto;">
				<h2>Новая заявка с лендинга</h2>
				%s
			</div>`,
			strings.Join(details, "\n"),
		),
	}

	resp, err := s.client.Emails.Send(params)
	if err == nil && resp != nil {
		fmt.Printf("[EMAIL] landing_contact_sent to=%s reply_to=%s from=%s resend_id=%s\n", s.contactTo, email, s.fromEmail, resp.Id)
	}
	return err
}
