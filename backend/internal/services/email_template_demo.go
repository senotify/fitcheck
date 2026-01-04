package services

import (
	"fmt"
	"os"
)

// DemoEmailTemplates generates sample email templates for testing and review
func DemoEmailTemplates() {
	emailService := NewEmailService(EmailServiceConfig{
		BaseURL: "http://localhost:8080",
		Enabled: false,
	})

	resultURL := "http://localhost:8080/api/result-link/abc123xyz789token"
	jobID := "demo-job-12345"

	// Generate HTML template
	htmlBody := emailService.buildHTMLEmailBody(resultURL, jobID)
	
	// Generate plain text template
	plainBody := emailService.buildPlainTextEmailBody(resultURL, jobID)

	// Write HTML template to file
	htmlFile := "email_template_demo.html"
	err := os.WriteFile(htmlFile, []byte(htmlBody), 0644)
	if err != nil {
		fmt.Printf("Error writing HTML template: %v\n", err)
	} else {
		fmt.Printf("HTML email template written to: %s\n", htmlFile)
	}

	// Write plain text template to file
	txtFile := "email_template_demo.txt"
	err = os.WriteFile(txtFile, []byte(plainBody), 0644)
	if err != nil {
		fmt.Printf("Error writing plain text template: %v\n", err)
	} else {
		fmt.Printf("Plain text email template written to: %s\n", txtFile)
	}

	// Print to console for quick review
	fmt.Println("\n=== HTML EMAIL TEMPLATE ===")
	fmt.Println(htmlBody)
	fmt.Println("\n=== PLAIN TEXT EMAIL TEMPLATE ===")
	fmt.Println(plainBody)
}
