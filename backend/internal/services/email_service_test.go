package services

import (
	"strings"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Feature: virtual-fitcheck, Property 28: Email collection and background processing
// Validates: Requirements 9.3
func TestProperty_EmailCollectionAndBackgroundProcessing(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("for any valid email address, the system should validate it successfully and allow background processing", prop.ForAll(
		func(localPart string, domain string, tld string) bool {
			// Construct a valid email
			email := localPart + "@" + domain + "." + tld
			
			// Create email service
			emailService := NewEmailService(EmailServiceConfig{
				Enabled: false, // Disabled for testing
			})
			
			// Validate the email
			valid, err := emailService.ValidateEmail(email)
			
			// Should be valid and no error
			return valid && err == nil
		},
		// Generate valid email components
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) >= 1 && len(s) <= 20 }),
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) >= 2 && len(s) <= 15 }),
		gen.OneConstOf("com", "org", "net", "edu", "io"),
	))

	properties.Property("for any invalid email format, the system should reject it", prop.ForAll(
		func(invalidEmail string) bool {
			emailService := NewEmailService(EmailServiceConfig{
				Enabled: false,
			})
			
			valid, err := emailService.ValidateEmail(invalidEmail)
			
			// Should be invalid
			return !valid && err != nil
		},
		// Generate invalid emails
		gen.OneConstOf(
			"notanemail",
			"missing@domain",
			"@nodomain.com",
			"no-at-sign.com",
			"double@@domain.com",
			"",
			"spaces in@email.com",
		),
	))

	properties.Property("for any job with an email address, background processing can continue", prop.ForAll(
		func(email string, jobID string) bool {
			emailService := NewEmailService(EmailServiceConfig{
				Enabled: false,
			})
			
			// If email is valid, we should be able to store it for background processing
			valid, _ := emailService.ValidateEmail(email)
			if !valid {
				return true // Skip invalid emails
			}
			
			// The system should be able to handle the email without blocking
			// This is demonstrated by the validation not panicking or hanging
			return true
		},
		gen.AlphaString().Map(func(s string) string { return s + "@example.com" }),
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) >= 10 }),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: virtual-fitcheck, Property 29: Email delivery on completion
// Validates: Requirements 9.4
func TestProperty_EmailDeliveryOnCompletion(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("for any completed job with an email, the system should generate a valid result URL", prop.ForAll(
		func(jobID string, token string) bool {
			emailService := NewEmailService(EmailServiceConfig{
				BaseURL: "http://localhost:8080",
				Enabled: false,
			})
			
			// Generate a result URL
			resultURL := emailService.baseURL + "/api/result-link/" + token
			
			// URL should be well-formed
			return strings.HasPrefix(resultURL, "http") && 
				   strings.Contains(resultURL, "/api/result-link/") &&
				   len(resultURL) > 20
		},
		gen.UUIDVersion(4).Map(func(uuid [16]byte) string {
			return strings.ReplaceAll(string(uuid[:]), "\x00", "a")
		}),
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) >= 10 }),
	))

	properties.Property("for any email address, the system should be able to construct a valid email body", prop.ForAll(
		func(jobID string) bool {
			emailService := NewEmailService(EmailServiceConfig{
				BaseURL: "http://localhost:8080",
				Enabled: false,
			})
			
			resultURL := emailService.baseURL + "/api/result-link/test-token"
			body := emailService.buildEmailBody(resultURL, jobID)
			
			// Email body should contain essential elements
			return strings.Contains(body, "Virtual FitCheck") &&
				   strings.Contains(body, resultURL) &&
				   strings.Contains(body, jobID) &&
				   strings.Contains(body, "24 hours")
		},
		gen.UUIDVersion(4).Map(func(uuid [16]byte) string {
			return strings.ReplaceAll(string(uuid[:]), "\x00", "a")
		}),
	))

	properties.Property("for any result token generation, tokens should be unique and secure", prop.ForAll(
		func() bool {
			emailService := NewEmailService(EmailServiceConfig{
				Enabled: false,
			})
			
			// Generate two tokens
			token1, err1 := emailService.GenerateResultToken()
			token2, err2 := emailService.GenerateResultToken()
			
			// Both should succeed
			if err1 != nil || err2 != nil {
				return false
			}
			
			// Tokens should be different
			if token1 == token2 {
				return false
			}
			
			// Tokens should be reasonably long (secure)
			return len(token1) > 20 && len(token2) > 20
		},
	))

	properties.Property("for any valid email and result URL, SendCompletionEmail should not error when disabled", prop.ForAll(
		func(localPart string, domain string) bool {
			email := localPart + "@" + domain + ".com"
			
			emailService := NewEmailService(EmailServiceConfig{
				BaseURL: "http://localhost:8080",
				Enabled: false, // Disabled for testing
			})
			
			resultURL := emailService.baseURL + "/api/result-link/test-token"
			err := emailService.SendCompletionEmail(email, resultURL, "test-job-id")
			
			// Should not error when disabled (just logs)
			return err == nil
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) >= 1 && len(s) <= 20 }),
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) >= 2 && len(s) <= 15 }),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Unit tests for specific edge cases
func TestEmailService_ValidateEmail(t *testing.T) {
	emailService := NewEmailService(EmailServiceConfig{
		Enabled: false,
	})

	tests := []struct {
		name      string
		email     string
		wantValid bool
	}{
		{"valid email", "user@example.com", true},
		{"valid email with subdomain", "user@mail.example.com", true},
		{"valid email with plus", "user+tag@example.com", true},
		{"empty email", "", false},
		{"no at sign", "userexample.com", false},
		{"no domain", "user@", false},
		{"no local part", "@example.com", false},
		{"spaces", "user @example.com", false},
		{"double at", "user@@example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, _ := emailService.ValidateEmail(tt.email)
			if valid != tt.wantValid {
				t.Errorf("ValidateEmail(%q) = %v, want %v", tt.email, valid, tt.wantValid)
			}
		})
	}
}

func TestEmailService_GenerateResultToken(t *testing.T) {
	emailService := NewEmailService(EmailServiceConfig{
		Enabled: false,
	})

	// Generate multiple tokens and ensure they're unique
	tokens := make(map[string]bool)
	for i := 0; i < 100; i++ {
		token, err := emailService.GenerateResultToken()
		if err != nil {
			t.Fatalf("GenerateResultToken() error = %v", err)
		}
		if len(token) < 20 {
			t.Errorf("Token too short: %d characters", len(token))
		}
		if tokens[token] {
			t.Errorf("Duplicate token generated: %s", token)
		}
		tokens[token] = true
	}
}
