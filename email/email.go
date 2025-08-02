package email

import (
	"bytes"
	"fmt"
	"html/template"
	"net/smtp"
	"os"
	"path/filepath"
)

type EmailConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

type EmailData struct {
	To           []string
	Subject      string
	TemplateName string
	Data         interface{}
}

var config EmailConfig

// InitEmailService initializes the email service with configuration
func InitEmailService() error {
	config = EmailConfig{
		Host:     os.Getenv("EMAIL_HOST"),
		Port:     os.Getenv("EMAIL_PORT"),
		Username: os.Getenv("EMAIL_USERNAME"),
		Password: os.Getenv("EMAIL_PASSWORD"),
		From:     os.Getenv("EMAIL_FROM"),
	}

	// Validate required configuration
	if config.Host == "" || config.Port == "" || config.Username == "" || config.Password == "" || config.From == "" {
		return fmt.Errorf("email configuration is incomplete, check environment variables")
	}

	return nil
}

// SendEmail sends an email using the provided template and data
func SendEmail(data EmailData) error {
	// Load the template
	templatePath := filepath.Join("templates", data.TemplateName+".html")
	t, err := template.ParseFiles(templatePath)
	if err != nil {
		return fmt.Errorf("failed to parse email template: %w", err)
	}

	// Execute the template with the provided data
	var body bytes.Buffer
	if err := t.Execute(&body, data.Data); err != nil {
		return fmt.Errorf("failed to execute email template: %w", err)
	}

	// Set up email headers
	headers := make(map[string]string)
	headers["From"] = config.From
	headers["To"] = data.To[0]
	headers["Subject"] = data.Subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=\"utf-8\""

	// Construct message
	message := ""
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + body.String()

	// Connect to SMTP server and send email
	auth := smtp.PlainAuth("", config.Username, config.Password, config.Host)
	addr := config.Host + ":" + config.Port

	if err := smtp.SendMail(addr, auth, config.From, data.To, []byte(message)); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

// SendGeneralMessage sends a general message email
func SendGeneralMessage(to []string, subject, message string, recipientName string) error {
	data := EmailData{
		To:           to,
		Subject:      subject,
		TemplateName: "general",
		Data: map[string]interface{}{
			"RecipientName": recipientName,
			"Message":       message,
		},
	}
	return SendEmail(data)
}

// SendBill sends a bill email
func SendBill(to []string, subject string, billData map[string]interface{}) error {
	data := EmailData{
		To:           to,
		Subject:      subject,
		TemplateName: "bill",
		Data:         billData,
	}
	return SendEmail(data)
}

// SendPaymentSlip sends a payment slip email
func SendPaymentSlip(to []string, subject string, paymentData map[string]interface{}) error {
	data := EmailData{
		To:           to,
		Subject:      subject,
		TemplateName: "payment",
		Data:         paymentData,
	}
	return SendEmail(data)
}
