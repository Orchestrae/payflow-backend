package email

import (
	"fmt"

	"github.com/matcornic/hermes/v2"
)

// PayslipNotification generates a payslip ready notification email.
func PayslipNotification(employeeName, period, grossPay, netPay, downloadURL string) hermes.Email {
	return hermes.Email{
		Body: hermes.Body{
			Name: employeeName,
			Intros: []string{
				fmt.Sprintf("Your payslip for %s is ready.", period),
			},
			Table: hermes.Table{
				Data: [][]hermes.Entry{
					{
						{Key: "Description", Value: "Gross Pay"},
						{Key: "Amount (NGN)", Value: grossPay},
					},
					{
						{Key: "Description", Value: "Net Pay"},
						{Key: "Amount (NGN)", Value: netPay},
					},
				},
			},
			Actions: []hermes.Action{
				{
					Instructions: "Download your full payslip:",
					Button: hermes.Button{
						Color: "#22BC66",
						Text:  "Download Payslip",
						Link:  downloadURL,
					},
				},
			},
		},
	}
}

// ApprovalRequest generates a payroll approval request email.
func ApprovalRequest(approverName, period string, employeeCount int, totalCost, approveURL string) hermes.Email {
	return hermes.Email{
		Body: hermes.Body{
			Name: approverName,
			Intros: []string{
				fmt.Sprintf("A payroll run for %s requires your approval.", period),
			},
			Table: hermes.Table{
				Data: [][]hermes.Entry{
					{
						{Key: "Detail", Value: "Period"},
						{Key: "Value", Value: period},
					},
					{
						{Key: "Detail", Value: "Employees"},
						{Key: "Value", Value: fmt.Sprintf("%d", employeeCount)},
					},
					{
						{Key: "Detail", Value: "Total Cost"},
						{Key: "Value", Value: fmt.Sprintf("NGN %s", totalCost)},
					},
				},
			},
			Actions: []hermes.Action{
				{
					Instructions: "Review and approve the payroll run:",
					Button: hermes.Button{
						Color: "#22BC66",
						Text:  "Review Payroll",
						Link:  approveURL,
					},
				},
			},
		},
	}
}

// RejectionNotification generates a payroll rejection notification email.
func RejectionNotification(operatorName, period, reason string) hermes.Email {
	return hermes.Email{
		Body: hermes.Body{
			Name: operatorName,
			Intros: []string{
				fmt.Sprintf("The payroll run for %s has been rejected.", period),
			},
			Dictionary: []hermes.Entry{
				{Key: "Period", Value: period},
				{Key: "Reason", Value: reason},
			},
			Outros: []string{
				"Please review the feedback and resubmit the payroll when ready.",
			},
		},
	}
}

// PasswordResetEmail generates a password reset email.
func PasswordResetEmail(userName, resetURL string) hermes.Email {
	return hermes.Email{
		Body: hermes.Body{
			Name: userName,
			Intros: []string{
				"You have requested to reset your password.",
			},
			Actions: []hermes.Action{
				{
					Instructions: "Click the button below to reset your password. This link expires in 1 hour.",
					Button: hermes.Button{
						Color: "#DC4D2F",
						Text:  "Reset Password",
						Link:  resetURL,
					},
				},
			},
			Outros: []string{
				"If you did not request a password reset, please ignore this email.",
			},
		},
	}
}

// UserInvitationEmail generates a user invitation email.
func UserInvitationEmail(businessName, role, inviteURL string) hermes.Email {
	return hermes.Email{
		Body: hermes.Body{
			Name: "there",
			Intros: []string{
				fmt.Sprintf("You have been invited to join %s on PayFlow as %s.", businessName, role),
				"PayFlow is an automated payroll platform that handles salary calculation, tax compliance, and payment disbursement.",
			},
			Actions: []hermes.Action{
				{
					Instructions: "Click the button below to accept your invitation and set your password:",
					Button: hermes.Button{
						Color: "#22BC66",
						Text:  "Accept Invitation",
						Link:  inviteURL,
					},
				},
			},
			Outros: []string{
				"This invitation link is valid for 7 days.",
			},
		},
	}
}

// LowBalanceAlert generates a low wallet balance alert email.
func LowBalanceAlert(adminName, businessName, currentBalance, topUpURL string) hermes.Email {
	return hermes.Email{
		Body: hermes.Body{
			Name: adminName,
			Intros: []string{
				fmt.Sprintf("Your PayFlow wallet balance for %s is running low.", businessName),
			},
			Dictionary: []hermes.Entry{
				{Key: "Current Balance", Value: fmt.Sprintf("NGN %s", currentBalance)},
			},
			Actions: []hermes.Action{
				{
					Instructions: "Top up your wallet to ensure payroll runs smoothly:",
					Button: hermes.Button{
						Color: "#F2994A",
						Text:  "Top Up Wallet",
						Link:  topUpURL,
					},
				},
			},
		},
	}
}

// ApprovalConfirmation generates a payroll approval confirmation email.
func ApprovalConfirmation(operatorName, period, scheduledDate string) hermes.Email {
	return hermes.Email{
		Body: hermes.Body{
			Name: operatorName,
			Intros: []string{
				fmt.Sprintf("The payroll run for %s has been approved.", period),
			},
			Dictionary: []hermes.Entry{
				{Key: "Period", Value: period},
				{Key: "Scheduled Processing", Value: scheduledDate},
			},
			Outros: []string{
				"Salaries will be disbursed on the scheduled date.",
			},
		},
	}
}
