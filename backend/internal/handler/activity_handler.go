package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// ActivityHandler handles user-facing activity endpoints.
type ActivityHandler struct {
	activityService      *service.UserActivityService
	firstRechargeService *service.FirstRechargeActivityService
}

func NewActivityHandler(activityService *service.UserActivityService, firstRechargeService *service.FirstRechargeActivityService) *ActivityHandler {
	return &ActivityHandler{
		activityService:      activityService,
		firstRechargeService: firstRechargeService,
	}
}

// List returns all activities currently visible to the authenticated user.
// GET /api/v1/activities
func (h *ActivityHandler) List(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	activities, err := h.activityService.ListForUser(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, activities)
}

// MarkViewed records that the user entered the activity center and viewed an activity.
// POST /api/v1/activities/:id/view
func (h *ActivityHandler) MarkViewed(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	if err := h.activityService.MarkViewed(c.Request.Context(), subject.UserID, c.Param("id")); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "ok"})
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
