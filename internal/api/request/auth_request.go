package request

import "time"

type RegisterRequest struct {
	BusinessName      string    `json:"business_name" validate:"required"`
	Email             string    `json:"email" validate:"required,email"`
	Password          string    `json:"password" validate:"required,min=8"`
	RCNumber          string    `json:"rc_number" validate:"required"`
	IncorporationDate time.Time `json:"incorporation_date" validate:"required"`
	DirectorBVN       string    `json:"director_bvn" validate:"required,len=11"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}
