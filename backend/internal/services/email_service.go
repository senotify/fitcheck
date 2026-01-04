package services

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/smtp"
	"regexp"
	"strings"
	"time"
)

// EmailService handles email notifications for completed jobs
type EmailService struct {
	smtpHost     string
	smtpPort     string
	smtpUsername string
	smtpPassword string
	fromEmail    string
	baseURL      string
	enabled      bool
}

// EmailServiceConfig holds configuration for the email service
type EmailServiceConfig struct {
	SMTPHost     string
	SMTPPort     string
	SMTPUsername string
	SMTPPassword string
	FromEmail    string
	BaseURL      string
	Enabled      bool
}

// NewEmailService creates a new EmailService instance
func NewEmailService(config EmailServiceConfig) *EmailService {
	return &EmailService{
		smtpHost:     config.SMTPHost,
		smtpPort:     config.SMTPPort,
		smtpUsername: config.SMTPUsername,
		smtpPassword: config.SMTPPassword,
		fromEmail:    config.FromEmail,
		baseURL:      config.BaseURL,
		enabled:      config.Enabled,
	}
}

// ValidateEmail validates an email address format
func (es *EmailService) ValidateEmail(email string) (bool, error) {
	if email == "" {
		return false, fmt.Errorf("email address is required")
	}

	// Basic email regex pattern
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	
	if !emailRegex.MatchString(email) {
		return false, fmt.Errorf("invalid email format")
	}

	return true, nil
}

// GenerateResultToken generates a secure time-limited token for result access
func (es *EmailService) GenerateResultToken() (string, error) {
	// Generate 32 random bytes
	tokenBytes := make([]byte, 32)
	_, err := rand.Read(tokenBytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	// Encode to base64 URL-safe string
	token := base64.URLEncoding.EncodeToString(tokenBytes)
	return token, nil
}

// SendCompletionEmail sends an email notification when a job completes
func (es *EmailService) SendCompletionEmail(email string, resultURL string, jobID string) error {
	if !es.enabled {
		// If email service is disabled, just log and return success
		fmt.Printf("Email service disabled. Would send email to %s for job %s\n", email, jobID)
		return nil
	}

	valid, err := es.ValidateEmail(email)
	if !valid {
		return err
	}

	// Construct the email
	subject := "Your Virtual FitCheck is Ready!"
	htmlBody := es.buildHTMLEmailBody(resultURL, jobID)
	plainBody := es.buildPlainTextEmailBody(resultURL, jobID)

	// Prepare multipart message with both plain text and HTML
	boundary := "boundary-virtualfitcheck-" + time.Now().Format("20060102150405")
	
	message := fmt.Sprintf("From: %s\r\n", es.fromEmail)
	message += fmt.Sprintf("To: %s\r\n", email)
	message += fmt.Sprintf("Subject: %s\r\n", subject)
	message += "MIME-Version: 1.0\r\n"
	message += fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"\r\n", boundary)
	message += "\r\n"
	
	// Plain text version
	message += fmt.Sprintf("--%s\r\n", boundary)
	message += "Content-Type: text/plain; charset=UTF-8\r\n"
	message += "Content-Transfer-Encoding: 7bit\r\n"
	message += "\r\n"
	message += plainBody
	message += "\r\n\r\n"
	
	// HTML version
	message += fmt.Sprintf("--%s\r\n", boundary)
	message += "Content-Type: text/html; charset=UTF-8\r\n"
	message += "Content-Transfer-Encoding: 7bit\r\n"
	message += "\r\n"
	message += htmlBody
	message += "\r\n\r\n"
	
	// End boundary
	message += fmt.Sprintf("--%s--\r\n", boundary)

	// Connect to SMTP server and send
	auth := smtp.PlainAuth("", es.smtpUsername, es.smtpPassword, es.smtpHost)
	addr := fmt.Sprintf("%s:%s", es.smtpHost, es.smtpPort)

	err = smtp.SendMail(addr, auth, es.fromEmail, []string{email}, []byte(message))
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

// buildHTMLEmailBody constructs the HTML email body
func (es *EmailService) buildHTMLEmailBody(resultURL string, jobID string) string {
	expirationTime := time.Now().Add(24 * time.Hour).Format("January 2, 2006 at 3:04 PM MST")
	
	html := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Your Virtual FitCheck is Ready!</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
    <div style="background-color: #f8f9fa; border-radius: 10px; padding: 30px; margin-bottom: 20px;">
        <h1 style="color: #2c3e50; margin-top: 0;">Your Virtual FitCheck is Ready! 🎉</h1>
        <p style="font-size: 16px; color: #555;">
            Great news! Your virtual try-on has been processed and is ready to view.
        </p>
    </div>
    
    <div style="background-color: #fff; border: 1px solid #e0e0e0; border-radius: 10px; padding: 30px; margin-bottom: 20px;">
        <h2 style="color: #2c3e50; margin-top: 0;">View Your Result</h2>
        <p style="font-size: 14px; color: #666; margin-bottom: 20px;">
            Click the button below to see how the shirt looks on you:
        </p>
        <div style="text-align: center; margin: 30px 0;">
            <a href="` + resultURL + `" 
               style="display: inline-block; background-color: #007bff; color: white; padding: 15px 40px; text-decoration: none; border-radius: 5px; font-size: 16px; font-weight: bold;">
                View My Result
            </a>
        </div>
        <p style="font-size: 12px; color: #999; margin-top: 20px;">
            Or copy and paste this link into your browser:<br>
            <a href="` + resultURL + `" style="color: #007bff; word-break: break-all;">` + resultURL + `</a>
        </p>
    </div>
    
    <div style="background-color: #fff3cd; border: 1px solid #ffc107; border-radius: 10px; padding: 20px; margin-bottom: 20px;">
        <p style="font-size: 14px; color: #856404; margin: 0;">
            ⏰ <strong>Important:</strong> This link will expire on <strong>` + expirationTime + `</strong> (24 hours from now).
        </p>
    </div>
    
    <div style="background-color: #f8f9fa; border-radius: 10px; padding: 20px; margin-bottom: 20px;">
        <p style="font-size: 12px; color: #666; margin: 0;">
            <strong>Job ID:</strong> ` + jobID + `<br>
            <strong>Processing completed:</strong> ` + time.Now().Format("January 2, 2006 at 3:04 PM MST") + `
        </p>
    </div>
    
    <div style="text-align: center; padding: 20px; color: #999; font-size: 12px;">
        <p>Thank you for using Virtual FitCheck!</p>
        <p>If you have any questions or need support, please contact us.</p>
    </div>
</body>
</html>
`
	
	return strings.TrimSpace(html)
}

// buildPlainTextEmailBody constructs the plain text email body as a fallback
func (es *EmailService) buildPlainTextEmailBody(resultURL string, jobID string) string {
	expirationTime := time.Now().Add(24 * time.Hour).Format("January 2, 2006 at 3:04 PM MST")
	completionTime := time.Now().Format("January 2, 2006 at 3:04 PM MST")
	
	plainText := fmt.Sprintf(`Your Virtual FitCheck is Ready!

Great news! Your virtual try-on has been processed and is ready to view.

VIEW YOUR RESULT
================
Click or copy the link below to see how the shirt looks on you:

%s

IMPORTANT NOTICE
================
This link will expire on %s (24 hours from now).

JOB DETAILS
===========
Job ID: %s
Processing completed: %s

Thank you for using Virtual FitCheck!
If you have any questions or need support, please contact us.

---
This is an automated message. Please do not reply to this email.
`, resultURL, expirationTime, jobID, completionTime)
	
	return strings.TrimSpace(plainText)
}
