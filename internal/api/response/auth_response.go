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
