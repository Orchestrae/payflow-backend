package response

type UserResponse struct {
	ID         uint   `json:"id"`
	Email      string `json:"email"`
	Role       string `json:"role"`
	BusinessID uint   `json:"business_id"`
}

type LoginResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}

type BusinessRegistrationResponse struct {
	User             UserResponse             `json:"user"`
	CorporateAccount CorporateAccountResponse `json:"corporate_account"`
}

type CorporateAccountResponse struct {
	AccountNumber string `json:"account_number"`
	AccountName   string `json:"account_name"`
}
