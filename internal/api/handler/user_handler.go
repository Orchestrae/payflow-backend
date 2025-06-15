// internal/api/handler/user_handler.go
package handler

import (
	"net/http"
	"payflow/internal/api/middleware"
	"payflow/internal/api/response"
	"payflow/internal/domain"
	"payflow/internal/service"
	"payflow/pkg/utils"
	"strconv"
)

type UserHandler struct {
	userService service.UserService
}

func NewUserHandler(svc service.UserService) *UserHandler {
	return &UserHandler{userService: svc}
}

// GetCurrentUser handles GET /users/me
// A utility endpoint for the frontend to get the logged-in user's details.
func (h *UserHandler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(middleware.UserClaimsKey).(*utils.Claims)
	if !ok {
		response.RespondWithError(w, domain.ErrInternalServer)
		return
	}

	userID, err := strconv.ParseUint(claims.UserID, 10, 32)
	if err != nil {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	user, err := h.userService.GetUserByID(r.Context(), uint(userID))
	if err != nil {
		response.RespondWithError(w, err)
		return
	}

	// Use the same UserResponse DTO from the auth response
	resp := response.UserResponse{
		ID:         user.ID,
		Email:      user.Email,
		Role:       string(user.Role),
		BusinessID: user.BusinessID,
	}

	response.RespondWithJSON(w, http.StatusOK, resp)
}

// TODO: Implement other user management handlers as needed
// - InviteUser (POST /users/invite)
// - ListUsersByBusiness (GET /users)
// - UpdateUserRole (PATCH /users/{userID}/role)
