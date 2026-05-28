package handler

import (
	"net/http"
	"strconv"

	"payflow/internal/api/middleware"
	"payflow/internal/api/response"
	"payflow/internal/domain"
	"payflow/internal/service"
)

// AuditHandler handles audit log endpoints.
type AuditHandler struct {
	auditService service.AuditService
}

// NewAuditHandler creates a new audit handler.
func NewAuditHandler(svc service.AuditService) *AuditHandler {
	return &AuditHandler{auditService: svc}
}

// ListAuditLogs handles GET /v1/audit-logs
func (h *AuditHandler) ListAuditLogs(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaimsFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, domain.ErrUnauthorized)
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	logs, total, err := h.auditService.ListByBusiness(r.Context(), claims.BusinessID, page, limit)
	if err != nil {
		response.RespondWithError(w, err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"data":  logs,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}
