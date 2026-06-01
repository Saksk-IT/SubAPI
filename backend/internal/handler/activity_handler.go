package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// ActivityHandler handles user-facing activity endpoints.
type ActivityHandler struct {
	firstRechargeService *service.FirstRechargeActivityService
}

func NewActivityHandler(firstRechargeService *service.FirstRechargeActivityService) *ActivityHandler {
	return &ActivityHandler{firstRechargeService: firstRechargeService}
}

// GetFirstRechargeStatus returns the authenticated user's first recharge status.
// GET /api/v1/activities/first-recharge/status
func (h *ActivityHandler) GetFirstRechargeStatus(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	status, err := h.firstRechargeService.GetStatus(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, status)
}

// DismissFirstRechargePopup records that the user manually closed the popup.
// POST /api/v1/activities/first-recharge/dismiss-popup
func (h *ActivityHandler) DismissFirstRechargePopup(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	if err := h.firstRechargeService.DismissPopup(c.Request.Context(), subject.UserID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "ok"})
}
