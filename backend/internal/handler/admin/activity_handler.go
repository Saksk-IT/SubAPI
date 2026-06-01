package admin

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// ActivityHandler handles admin activity management endpoints.
type ActivityHandler struct {
	firstRechargeService *service.FirstRechargeActivityService
	adminService         service.AdminService
}

func NewActivityHandler(firstRechargeService *service.FirstRechargeActivityService, adminService service.AdminService) *ActivityHandler {
	return &ActivityHandler{
		firstRechargeService: firstRechargeService,
		adminService:         adminService,
	}
}

type UpdateFirstRechargeRequest struct {
	Enabled          bool                              `json:"enabled"`
	EligibilityScope string                            `json:"eligibility_scope"`
	Offers           []service.FirstRechargeOfferInput `json:"offers"`
}

type AddFirstRechargeUserRequest struct {
	UserID int64 `json:"user_id"`
}

type FirstRechargeUserSummary struct {
	ID       int64  `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
}

// GetFirstRecharge returns first recharge config and offers.
// GET /api/v1/admin/activities/first-recharge
func (h *ActivityHandler) GetFirstRecharge(c *gin.Context) {
	config, err := h.firstRechargeService.GetAdminConfig(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, config)
}

// UpdateFirstRecharge updates first recharge config and offers.
// PUT /api/v1/admin/activities/first-recharge
func (h *ActivityHandler) UpdateFirstRecharge(c *gin.Context) {
	var req UpdateFirstRechargeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	config, err := h.firstRechargeService.UpdateAdminConfig(c.Request.Context(), service.UpdateFirstRechargeConfigInput{
		Enabled:          req.Enabled,
		EligibilityScope: req.EligibilityScope,
		Offers:           req.Offers,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, config)
}

// ListFirstRechargeUsers lists users explicitly assigned first recharge rights.
// GET /api/v1/admin/activities/first-recharge/users
func (h *ActivityHandler) ListFirstRechargeUsers(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	items, result, err := h.firstRechargeService.ListSpecifiedUsers(
		c.Request.Context(),
		pagination.PaginationParams{Page: page, PageSize: pageSize},
		c.Query("search"),
	)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, result.Total, page, pageSize)
}

// AddFirstRechargeUser adds one user to the specified-user eligibility list.
// POST /api/v1/admin/activities/first-recharge/users
func (h *ActivityHandler) AddFirstRechargeUser(c *gin.Context) {
	var req AddFirstRechargeUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	var actorID *int64
	if subject, ok := middleware2.GetAuthSubjectFromContext(c); ok && subject.UserID > 0 {
		id := subject.UserID
		actorID = &id
	}
	if err := h.firstRechargeService.AddSpecifiedUser(c.Request.Context(), req.UserID, actorID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"user_id": req.UserID})
}

// RemoveFirstRechargeUser removes one user from the specified-user eligibility list.
// DELETE /api/v1/admin/activities/first-recharge/users/:user_id
func (h *ActivityHandler) RemoveFirstRechargeUser(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil || userID <= 0 {
		response.BadRequest(c, "Invalid user_id")
		return
	}
	if err := h.firstRechargeService.RemoveSpecifiedUser(c.Request.Context(), userID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"user_id": userID})
}

// LookupUsers searches users for the add specified-user selector.
// GET /api/v1/admin/activities/first-recharge/users/lookup?q=
func (h *ActivityHandler) LookupUsers(c *gin.Context) {
	keyword := c.Query("q")
	if keyword == "" {
		response.Success(c, []FirstRechargeUserSummary{})
		return
	}
	users, _, err := h.adminService.ListUsers(c.Request.Context(), 1, 20, service.UserListFilters{Search: keyword}, "email", "asc")
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	result := make([]FirstRechargeUserSummary, 0, len(users))
	for _, u := range users {
		result = append(result, FirstRechargeUserSummary{ID: u.ID, Email: u.Email, Username: u.Username})
	}
	response.Success(c, result)
}
