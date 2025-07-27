// internal/api/handler/auth_handler.go
package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"payflow/internal/api/request"
	"payflow/internal/api/response"
	"payflow/internal/domain"
	"payflow/internal/service"

	"github.com/go-playground/validator/v10"
)

type AuthHandler struct {
	authService service.AuthService
	validate    *validator.Validate
}

func NewAuthHandler(authSvc service.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authSvc,
		validate:    validator.New(),
	}
}

func (h *AuthHandler) RegisterBusiness(w http.ResponseWriter, r *http.Request) {
	var req request.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("Failed to decode request body", "error", err)
		response.RespondWithError(w, domain.ErrValidationFailed) // Bad JSON format
		return
	}

	if err := h.validate.Struct(req); err != nil {
		// Handle validation errors more gracefully in a real app
		slog.Error("Validation failed", "error", err)
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	user, corporateAccount, err := h.authService.RegisterBusiness(
		r.Context(),
		req.BusinessName,
		req.Email,
		req.Password,
		req.RCNumber,
		req.IncorporationDate,
		req.DirectorBVN,
	)
	if err != nil {
		slog.Error("Failed to register business", "error", err)
		response.RespondWithError(w, err)
		return
	}

	resp := response.BusinessRegistrationResponse{
		User: response.UserResponse{
			ID:         user.ID,
			Email:      user.Email,
			Role:       string(user.Role),
			BusinessID: user.BusinessID,
		},
		CorporateAccount: response.CorporateAccountResponse{
			AccountNumber: corporateAccount.AccountNumber,
			AccountName:   corporateAccount.AccountName,
		},
	}
	response.RespondWithJSON(w, http.StatusCreated, resp)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req request.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	token, user, err := h.authService.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		response.RespondWithError(w, err)
		return
	}
	resp := response.LoginResponse{
		Token: token,
		User: response.UserResponse{
			ID:         user.ID,
			Email:      user.Email,
			Role:       string(user.Role),
			BusinessID: user.BusinessID,
		},
	}
	response.RespondWithJSON(w, http.StatusOK, resp)
}
