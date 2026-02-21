// internal/api/handler/user_handler.go
package handler

import (
	"net/http"
	"payflow/internal/api/middleware"
	"payflow/internal/api/response"
	"payflow/internal/domain"
	"payflow/internal/service"
)

type UserHandler struct {
	userService service.UserService
}

func NewUserHandler(svc service.UserService) *UserHandler {
	return &UserHandler{userService: svc}
}

// GetCurrentUser handles GET /users/me
func (h *UserHandler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaimsFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, domain.ErrUnauthorized)
		return
	}

	user, err := h.userService.GetUserByID(r.Context(), claims.UserID)
	if err != nil {
		response.RespondWithError(w, err)
		return
	}

	resp := response.UserResponse{
		ID:         user.ID,
		Email:      user.Email,
		Role:       string(user.Role),
		BusinessID: user.BusinessID,
	}

	response.RespondWithJSON(w, http.StatusOK, resp)
}
