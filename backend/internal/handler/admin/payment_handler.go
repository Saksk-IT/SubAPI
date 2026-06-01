package admin

import (
	"strconv"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// PaymentHandler handles admin payment management.
type PaymentHandler struct {
	paymentService *service.PaymentService
	configService  *service.PaymentConfigService
}

// NewPaymentHandler creates a new admin PaymentHandler.
func NewPaymentHandler(paymentService *service.PaymentService, configService *service.PaymentConfigService) *PaymentHandler {
	return &PaymentHandler{
		paymentService: paymentService,
		configService:  configService,
	}
}

// --- Dashboard ---

// GetDashboard returns payment dashboard statistics.
// GET /api/v1/admin/payment/dashboard
func (h *PaymentHandler) GetDashboard(c *gin.Context) {
	days := 30
	if d := c.Query("days"); d != "" {
		if v, err := strconv.Atoi(d); err == nil && v > 0 {
			days = v
		}
	}
	activityType, ok := parsePaymentActivityType(c)
	if !ok {
		return
	}
	stats, err := h.paymentService.GetDashboardStatsWithParams(c.Request.Context(), service.DashboardStatsParams{
		Days:         days,
		ActivityType: activityType,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, stats)
}

// --- Orders ---

// ListOrders returns a paginated list of all payment orders.
// GET /api/v1/admin/payment/orders
func (h *PaymentHandler) ListOrders(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	var userID int64
	if uid := c.Query("user_id"); uid != "" {
		if v, err := strconv.ParseInt(uid, 10, 64); err == nil {
			userID = v
		}
	}
	activityType, ok := parsePaymentActivityType(c)
	if !ok {
		return
	}
	orders, total, err := h.paymentService.AdminListOrders(c.Request.Context(), userID, service.OrderListParams{
		Page:         page,
		PageSize:     pageSize,
		Status:       c.Query("status"),
		OrderType:    c.Query("order_type"),
		PaymentType:  c.Query("payment_type"),
		ActivityType: activityType,
		Keyword:      c.Query("keyword"),
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, sanitizeAdminPaymentOrdersForResponse(orders), int64(total), page, pageSize)
}

// GetOrderDetail returns detailed information about a single order.
// GET /api/v1/admin/payment/orders/:id
func (h *PaymentHandler) GetOrderDetail(c *gin.Context) {
	orderID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	order, err := h.paymentService.GetOrderByID(c.Request.Context(), orderID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	auditLogs, _ := h.paymentService.GetOrderAuditLogs(c.Request.Context(), orderID)
	response.Success(c, gin.H{"order": sanitizeAdminPaymentOrderForResponse(order), "auditLogs": auditLogs})
}

// CancelOrder cancels a pending order (admin).
// POST /api/v1/admin/payment/orders/:id/cancel
func (h *PaymentHandler) CancelOrder(c *gin.Context) {
	orderID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	msg, err := h.paymentService.AdminCancelOrder(c.Request.Context(), orderID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": msg})
}

// RetryFulfillment retries fulfillment for a paid order.
// POST /api/v1/admin/payment/orders/:id/retry
func (h *PaymentHandler) RetryFulfillment(c *gin.Context) {
	orderID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.paymentService.RetryFulfillment(c.Request.Context(), orderID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "fulfillment retried"})
}

func sanitizeAdminPaymentOrdersForResponse(orders []*dbent.PaymentOrder) []*dbent.PaymentOrder {
	if len(orders) == 0 {
		return orders
	}
	out := make([]*dbent.PaymentOrder, 0, len(orders))
	for _, order := range orders {
		out = append(out, sanitizeAdminPaymentOrderForResponse(order))
	}
	return out
}

func parsePaymentActivityType(c *gin.Context) (string, bool) {
	activityType := c.Query("activity_type")
	if activityType == "" || activityType == service.FirstRechargeActivityType {
		return activityType, true
	}
	response.BadRequest(c, "Invalid activity_type")
	return "", false
}

func sanitizeAdminPaymentOrderForResponse(order *dbent.PaymentOrder) *dbent.PaymentOrder {
	if order == nil {
		return nil
	}
	cloned := *order
	cloned.ProviderSnapshot = nil
	return &cloned
}

// AdminProcessRefundRequest is the request body for admin refund processing.
type AdminProcessRefundRequest struct {
	Amount        float64 `json:"amount"`
	Reason        string  `json:"reason"`
	Force         bool    `json:"force"`
	DeductBalance bool    `json:"deduct_balance"`
}

// ProcessRefund processes a refund for an order (admin).
// POST /api/v1/admin/payment/orders/:id/refund
func (h *PaymentHandler) ProcessRefund(c *gin.Context) {
	orderID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	var req AdminProcessRefundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	plan, earlyResult, err := h.paymentService.PrepareRefund(c.Request.Context(), orderID, req.Amount, req.Reason, req.Force, req.DeductBalance)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if earlyResult != nil {
		response.Success(c, earlyResult)
		return
	}

	result, err := h.paymentService.ExecuteRefund(c.Request.Context(), plan)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

// --- Subscription Plans ---

type AdminPlanResult struct {
	ID            int64    `json:"id"`
	GroupID       int64    `json:"group_id"`
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	Price         float64  `json:"price"`
	OriginalPrice *float64 `json:"original_price,omitempty"`
	ValidityDays  int      `json:"validity_days"`
	ValidityUnit  string   `json:"validity_unit"`
	Features      string   `json:"features"`
	Tags          string   `json:"tags"`
	TotalQuota    *float64 `json:"total_quota,omitempty"`
	DailyQuota    *float64 `json:"daily_quota,omitempty"`
	DisplayNotes  string   `json:"display_notes"`
	ProductName   string   `json:"product_name"`
	ForSale       bool     `json:"for_sale"`
	SortOrder     int      `json:"sort_order"`
	SalesCount    int64    `json:"sales_count"`
}

type ProductSortOrderRequest struct {
	Updates []struct {
		ID        int64 `json:"id" binding:"required"`
		SortOrder int   `json:"sort_order"`
	} `json:"updates" binding:"required,min=1"`
}

func productSortUpdatesFromRequest(req ProductSortOrderRequest) []service.ProductSortOrderUpdate {
	updates := make([]service.ProductSortOrderUpdate, 0, len(req.Updates))
	for _, update := range req.Updates {
		updates = append(updates, service.ProductSortOrderUpdate{
			ID:        update.ID,
			SortOrder: update.SortOrder,
		})
	}
	return updates
}

// ListPlans returns all subscription plans.
// GET /api/v1/admin/payment/plans
func (h *PaymentHandler) ListPlans(c *gin.Context) {
	plans, err := h.configService.ListPlans(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	results, err := h.buildAdminPlanResults(c, plans)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, results)
}

// CreatePlan creates a new subscription plan.
// POST /api/v1/admin/payment/plans
func (h *PaymentHandler) CreatePlan(c *gin.Context) {
	var req service.CreatePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	plan, err := h.configService.CreatePlan(c.Request.Context(), req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	results, err := h.buildAdminPlanResults(c, []*dbent.SubscriptionPlan{plan})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, results[0])
}

// UpdatePlan updates an existing subscription plan.
// PUT /api/v1/admin/payment/plans/:id
func (h *PaymentHandler) UpdatePlan(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	var req service.UpdatePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	plan, err := h.configService.UpdatePlan(c.Request.Context(), id, req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	results, err := h.buildAdminPlanResults(c, []*dbent.SubscriptionPlan{plan})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, results[0])
}

func (h *PaymentHandler) UpdatePlanSortOrder(c *gin.Context) {
	var req ProductSortOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if err := h.configService.UpdatePlanSortOrders(c.Request.Context(), productSortUpdatesFromRequest(req)); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "sort order updated"})
}

// DeletePlan deletes a subscription plan.
// DELETE /api/v1/admin/payment/plans/:id
func (h *PaymentHandler) DeletePlan(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.configService.DeletePlan(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "deleted"})
}

func (h *PaymentHandler) buildAdminPlanResults(c *gin.Context, plans []*dbent.SubscriptionPlan) ([]AdminPlanResult, error) {
	displayInfo := h.configService.GetPlanDisplayInfoMap(c.Request.Context(), plans)
	salesCounts, err := h.configService.GetPlanSalesCountMap(c.Request.Context(), plans)
	if err != nil {
		return nil, err
	}
	out := make([]AdminPlanResult, 0, len(plans))
	for _, plan := range plans {
		display := displayInfo[plan.ID]
		out = append(out, AdminPlanResult{
			ID:            plan.ID,
			GroupID:       plan.GroupID,
			Name:          plan.Name,
			Description:   plan.Description,
			Price:         plan.Price,
			OriginalPrice: plan.OriginalPrice,
			ValidityDays:  plan.ValidityDays,
			ValidityUnit:  plan.ValidityUnit,
			Features:      plan.Features,
			Tags:          display.Tags,
			TotalQuota:    display.TotalQuota,
			DailyQuota:    display.DailyQuota,
			DisplayNotes:  display.DisplayNotes,
			ProductName:   plan.ProductName,
			ForSale:       plan.ForSale,
			SortOrder:     plan.SortOrder,
			SalesCount:    salesCounts[plan.ID],
		})
	}
	return out, nil
}

// --- Balance Products ---

func (h *PaymentHandler) ListBalanceProducts(c *gin.Context) {
	products, err := h.configService.ListBalanceProducts(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, products)
}

func (h *PaymentHandler) CreateBalanceProduct(c *gin.Context) {
	var req service.CreateBalanceProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	product, err := h.configService.CreateBalanceProduct(c.Request.Context(), req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, product)
}

func (h *PaymentHandler) UpdateBalanceProduct(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	var req service.UpdateBalanceProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	product, err := h.configService.UpdateBalanceProduct(c.Request.Context(), id, req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, product)
}

func (h *PaymentHandler) UpdateBalanceProductSortOrder(c *gin.Context) {
	var req ProductSortOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if err := h.configService.UpdateBalanceProductSortOrders(c.Request.Context(), productSortUpdatesFromRequest(req)); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "sort order updated"})
}

func (h *PaymentHandler) DeleteBalanceProduct(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.configService.DeleteBalanceProduct(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "deleted"})
}

// --- Provider Instances ---

// ListProviders returns all payment provider instances.
// GET /api/v1/admin/payment/providers
func (h *PaymentHandler) ListProviders(c *gin.Context) {
	providers, err := h.configService.ListProviderInstancesWithConfig(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, providers)
}

// CreateProvider creates a new payment provider instance.
// POST /api/v1/admin/payment/providers
func (h *PaymentHandler) CreateProvider(c *gin.Context) {
	var req service.CreateProviderInstanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	inst, err := h.configService.CreateProviderInstance(c.Request.Context(), req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	h.paymentService.RefreshProviders(c.Request.Context())
	response.Created(c, inst)
}

// UpdateProvider updates an existing payment provider instance.
// PUT /api/v1/admin/payment/providers/:id
func (h *PaymentHandler) UpdateProvider(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	var req service.UpdateProviderInstanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	inst, err := h.configService.UpdateProviderInstance(c.Request.Context(), id, req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	h.paymentService.RefreshProviders(c.Request.Context())
	response.Success(c, inst)
}

// DeleteProvider deletes a payment provider instance.
// DELETE /api/v1/admin/payment/providers/:id
func (h *PaymentHandler) DeleteProvider(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.configService.DeleteProviderInstance(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	h.paymentService.RefreshProviders(c.Request.Context())
	response.Success(c, gin.H{"message": "deleted"})
}

// parseIDParam parses an int64 path parameter.
// Returns the parsed ID and true on success; on failure it writes a BadRequest response and returns false.
func parseIDParam(c *gin.Context, paramName string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(paramName), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid "+paramName)
		return 0, false
	}
	return id, true
}

// --- Config ---

// GetConfig returns the payment configuration (admin view).
// GET /api/v1/admin/payment/config
func (h *PaymentHandler) GetConfig(c *gin.Context) {
	cfg, err := h.configService.GetPaymentConfig(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, cfg)
}

// UpdateConfig updates the payment configuration.
// PUT /api/v1/admin/payment/config
func (h *PaymentHandler) UpdateConfig(c *gin.Context) {
	var req service.UpdatePaymentConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if err := h.configService.UpdatePaymentConfig(c.Request.Context(), req); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "updated"})
}
