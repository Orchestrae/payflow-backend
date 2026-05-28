package email

import (
	"strings"
	"testing"

	"github.com/matcornic/hermes/v2"
)

func testHermes() hermes.Hermes {
	return hermes.Hermes{
		Product: hermes.Product{
			Name: "PayFlow",
			Link: "https://payflow.test",
		},
	}
}

func TestPayslipNotification(t *testing.T) {
	h := testHermes()
	email := PayslipNotification("John Doe", "January 2026", "300,000.00", "240,132.00", "https://payflow.test/payslips/1")

	html, err := h.GenerateHTML(email)
	if err != nil {
		t.Fatalf("failed to generate HTML: %v", err)
	}

	if !strings.Contains(html, "John Doe") {
		t.Error("HTML should contain employee name")
	}
	if !strings.Contains(html, "January 2026") {
		t.Error("HTML should contain period")
	}
	if !strings.Contains(html, "300,000.00") {
		t.Error("HTML should contain gross pay")
	}
	if !strings.Contains(html, "240,132.00") {
		t.Error("HTML should contain net pay")
	}
	if !strings.Contains(html, "Download Payslip") {
		t.Error("HTML should contain download button")
	}
}

func TestPasswordResetEmail(t *testing.T) {
	h := testHermes()
	email := PasswordResetEmail("Jane Smith", "https://payflow.test/reset?token=abc123")

	html, err := h.GenerateHTML(email)
	if err != nil {
		t.Fatalf("failed to generate HTML: %v", err)
	}

	if !strings.Contains(html, "Jane Smith") {
		t.Error("HTML should contain user name")
	}
	if !strings.Contains(html, "Reset Password") {
		t.Error("HTML should contain reset button")
	}
	if !strings.Contains(html, "1 hour") {
		t.Error("HTML should mention expiry")
	}
}

func TestUserInvitationEmail(t *testing.T) {
	h := testHermes()
	email := UserInvitationEmail("Acme Corp", "operator", "https://payflow.test/invite?token=xyz789")

	html, err := h.GenerateHTML(email)
	if err != nil {
		t.Fatalf("failed to generate HTML: %v", err)
	}

	if !strings.Contains(html, "Acme Corp") {
		t.Error("HTML should contain business name")
	}
	if !strings.Contains(html, "operator") {
		t.Error("HTML should contain role")
	}
	if !strings.Contains(html, "Accept Invitation") {
		t.Error("HTML should contain accept button")
	}
}

func TestApprovalRequest(t *testing.T) {
	h := testHermes()
	email := ApprovalRequest("Admin User", "February 2026", 50, "1,500,000.00", "https://payflow.test/approve/1")

	html, err := h.GenerateHTML(email)
	if err != nil {
		t.Fatalf("failed to generate HTML: %v", err)
	}

	if !strings.Contains(html, "February 2026") {
		t.Error("HTML should contain period")
	}
	if !strings.Contains(html, "50") {
		t.Error("HTML should contain employee count")
	}
	if !strings.Contains(html, "Review Payroll") {
		t.Error("HTML should contain review button")
	}
}

func TestRejectionNotification(t *testing.T) {
	h := testHermes()
	email := RejectionNotification("Op User", "March 2026", "Incorrect salary figures")

	html, err := h.GenerateHTML(email)
	if err != nil {
		t.Fatalf("failed to generate HTML: %v", err)
	}

	if !strings.Contains(html, "rejected") {
		t.Error("HTML should mention rejection")
	}
	if !strings.Contains(html, "Incorrect salary figures") {
		t.Error("HTML should contain rejection reason")
	}
}

func TestLowBalanceAlert(t *testing.T) {
	h := testHermes()
	email := LowBalanceAlert("Admin", "Acme Corp", "50,000.00", "https://payflow.test/wallet")

	html, err := h.GenerateHTML(email)
	if err != nil {
		t.Fatalf("failed to generate HTML: %v", err)
	}

	if !strings.Contains(html, "50,000.00") {
		t.Error("HTML should contain balance")
	}
	if !strings.Contains(html, "Top Up Wallet") {
		t.Error("HTML should contain top up button")
	}
}

func TestAllTemplatesGeneratePlainText(t *testing.T) {
	h := testHermes()
	templates := []struct {
		name  string
		email hermes.Email
	}{
		{"Payslip", PayslipNotification("Test", "Jan 2026", "100", "80", "http://test")},
		{"Reset", PasswordResetEmail("Test", "http://test")},
		{"Invite", UserInvitationEmail("Corp", "admin", "http://test")},
		{"Approval", ApprovalRequest("Test", "Jan 2026", 10, "500", "http://test")},
		{"Rejection", RejectionNotification("Test", "Jan 2026", "Bad data")},
		{"LowBalance", LowBalanceAlert("Test", "Corp", "100", "http://test")},
		{"Confirmation", ApprovalConfirmation("Test", "Jan 2026", "2026-01-15")},
	}

	for _, tt := range templates {
		t.Run(tt.name, func(t *testing.T) {
			text, err := h.GeneratePlainText(tt.email)
			if err != nil {
				t.Fatalf("failed to generate plain text: %v", err)
			}
			if len(text) == 0 {
				t.Error("plain text should not be empty")
			}
		})
	}
}
