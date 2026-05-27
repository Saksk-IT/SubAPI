package service

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/group"
	"github.com/Wei-Shaw/sub2api/ent/subscriptionplan"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

type planDerivedQuota struct {
	TotalQuota *float64
	DailyQuota *float64
}

// validatePlanRequired checks that all required fields for a plan are provided.
func validatePlanRequired(name string, groupID int64, price float64, validityDays int, validityUnit string, originalPrice *float64) error {
	if strings.TrimSpace(name) == "" {
		return infraerrors.BadRequest("PLAN_NAME_REQUIRED", "plan name is required")
	}
	if groupID <= 0 {
		return infraerrors.BadRequest("PLAN_GROUP_REQUIRED", "group is required")
	}
	if price <= 0 {
		return infraerrors.BadRequest("PLAN_PRICE_INVALID", "price must be > 0")
	}
	if validityDays <= 0 {
		return infraerrors.BadRequest("PLAN_VALIDITY_REQUIRED", "validity days must be > 0")
	}
	if originalPrice != nil && *originalPrice < 0 {
		return infraerrors.BadRequest("PLAN_ORIGINAL_PRICE_INVALID", "original price must be >= 0")
	}
	return nil
}

func validatePlanDisplayFields(tags string, features string, totalQuota *float64, dailyQuota *float64) error {
	if err := validateProductLines(tags, maxProductTags, maxProductTagLen, "PLAN_TAGS_INVALID"); err != nil {
		return err
	}
	if err := validateProductLines(features, maxProductFeatures, 160, "PLAN_FEATURES_INVALID"); err != nil {
		return err
	}
	if totalQuota != nil && *totalQuota < 0 {
		return infraerrors.BadRequest("PLAN_TOTAL_QUOTA_INVALID", "total quota must be >= 0")
	}
	if dailyQuota != nil && *dailyQuota < 0 {
		return infraerrors.BadRequest("PLAN_DAILY_QUOTA_INVALID", "daily quota must be >= 0")
	}
	return nil
}

func normalizePlanValidityUnit(unit string) string {
	normalized := strings.ToLower(strings.TrimSpace(unit))
	switch normalized {
	case "week", "weeks":
		return "weeks"
	case "month", "months":
		return "months"
	default:
		return "days"
	}
}

func positiveQuotaPtr(value *float64) *float64 {
	if value == nil || *value <= 0 {
		return nil
	}
	quota := *value
	return &quota
}

func roundQuota(value float64) float64 {
	return math.Round(value*100) / 100
}

func derivePlanQuotaFromGroup(group *dbent.Group, validityDays int, validityUnit string) planDerivedQuota {
	if group == nil || validityDays <= 0 {
		return planDerivedQuota{}
	}

	var cycleQuota *float64
	switch normalizePlanValidityUnit(validityUnit) {
	case "weeks":
		cycleQuota = positiveQuotaPtr(group.WeeklyLimitUsd)
	case "months":
		cycleQuota = positiveQuotaPtr(group.MonthlyLimitUsd)
	default:
		cycleQuota = positiveQuotaPtr(group.DailyLimitUsd)
	}

	var totalQuota *float64
	if cycleQuota != nil {
		total := roundQuota(*cycleQuota * float64(validityDays))
		totalQuota = &total
	}

	return planDerivedQuota{
		TotalQuota: totalQuota,
		DailyQuota: positiveQuotaPtr(group.DailyLimitUsd),
	}
}

func derivePlanValidityUnitFromGroup(group *dbent.Group) string {
	if group == nil {
		return "days"
	}
	if positiveQuotaPtr(group.MonthlyLimitUsd) != nil {
		return "months"
	}
	if positiveQuotaPtr(group.WeeklyLimitUsd) != nil {
		return "weeks"
	}
	if positiveQuotaPtr(group.DailyLimitUsd) != nil {
		return "days"
	}
	return "days"
}

// validatePlanPatch validates only the non-nil fields in a patch update.
func validatePlanPatch(req UpdatePlanRequest) error {
	if req.Name != nil && strings.TrimSpace(*req.Name) == "" {
		return infraerrors.BadRequest("PLAN_NAME_REQUIRED", "plan name is required")
	}
	if req.GroupID != nil && *req.GroupID <= 0 {
		return infraerrors.BadRequest("PLAN_GROUP_REQUIRED", "group is required")
	}
	if req.Price != nil && *req.Price <= 0 {
		return infraerrors.BadRequest("PLAN_PRICE_INVALID", "price must be > 0")
	}
	if req.ValidityDays != nil && *req.ValidityDays <= 0 {
		return infraerrors.BadRequest("PLAN_VALIDITY_REQUIRED", "validity days must be > 0")
	}
	if req.ValidityUnit != nil && strings.TrimSpace(*req.ValidityUnit) == "" {
		return infraerrors.BadRequest("PLAN_VALIDITY_UNIT_REQUIRED", "validity unit is required")
	}
	if req.OriginalPrice != nil && *req.OriginalPrice < 0 {
		return infraerrors.BadRequest("PLAN_ORIGINAL_PRICE_INVALID", "original price must be >= 0")
	}
	if req.Tags != nil {
		if err := validateProductLines(*req.Tags, maxProductTags, maxProductTagLen, "PLAN_TAGS_INVALID"); err != nil {
			return err
		}
	}
	if req.Features != nil {
		if err := validateProductLines(*req.Features, maxProductFeatures, 160, "PLAN_FEATURES_INVALID"); err != nil {
			return err
		}
	}
	if req.TotalQuota != nil && *req.TotalQuota < 0 {
		return infraerrors.BadRequest("PLAN_TOTAL_QUOTA_INVALID", "total quota must be >= 0")
	}
	if req.DailyQuota != nil && *req.DailyQuota < 0 {
		return infraerrors.BadRequest("PLAN_DAILY_QUOTA_INVALID", "daily quota must be >= 0")
	}
	return nil
}

// --- Plan CRUD ---

// PlanGroupInfo holds the group details needed for subscription plan display.
type PlanGroupInfo struct {
	Platform        string   `json:"platform"`
	Name            string   `json:"name"`
	RateMultiplier  float64  `json:"rate_multiplier"`
	DailyLimitUSD   *float64 `json:"daily_limit_usd"`
	WeeklyLimitUSD  *float64 `json:"weekly_limit_usd"`
	MonthlyLimitUSD *float64 `json:"monthly_limit_usd"`
	ModelScopes     []string `json:"supported_model_scopes"`
}

type PlanDisplayInfo struct {
	Tags         string   `json:"tags"`
	TotalQuota   *float64 `json:"total_quota,omitempty"`
	DailyQuota   *float64 `json:"daily_quota,omitempty"`
	DisplayNotes string   `json:"display_notes"`
}

// GetGroupPlatformMap returns a map of group_id → platform for the given plans.
func (s *PaymentConfigService) GetGroupPlatformMap(ctx context.Context, plans []*dbent.SubscriptionPlan) map[int64]string {
	info := s.GetGroupInfoMap(ctx, plans)
	m := make(map[int64]string, len(info))
	for id, gi := range info {
		m[id] = gi.Platform
	}
	return m
}

// GetGroupInfoMap returns a map of group_id → PlanGroupInfo for the given plans.
func (s *PaymentConfigService) GetGroupInfoMap(ctx context.Context, plans []*dbent.SubscriptionPlan) map[int64]PlanGroupInfo {
	ids := make([]int64, 0, len(plans))
	seen := make(map[int64]bool)
	for _, p := range plans {
		if !seen[p.GroupID] {
			seen[p.GroupID] = true
			ids = append(ids, p.GroupID)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	groups, err := s.entClient.Group.Query().Where(group.IDIn(ids...)).All(ctx)
	if err != nil {
		return nil
	}
	m := make(map[int64]PlanGroupInfo, len(groups))
	for _, g := range groups {
		m[int64(g.ID)] = PlanGroupInfo{
			Platform:        g.Platform,
			Name:            g.Name,
			RateMultiplier:  g.RateMultiplier,
			DailyLimitUSD:   g.DailyLimitUsd,
			WeeklyLimitUSD:  g.WeeklyLimitUsd,
			MonthlyLimitUSD: g.MonthlyLimitUsd,
			ModelScopes:     g.SupportedModelScopes,
		}
	}
	return m
}

func (s *PaymentConfigService) GetPlanDisplayInfoMap(ctx context.Context, plans []*dbent.SubscriptionPlan) map[int64]PlanDisplayInfo {
	ids := make([]int64, 0, len(plans))
	for _, p := range plans {
		ids = append(ids, p.ID)
	}
	if len(ids) == 0 {
		return nil
	}
	placeholders := make([]string, 0, len(ids))
	args := make([]any, 0, len(ids))
	for i, id := range ids {
		placeholders = append(placeholders, fmt.Sprintf("$%d", i+1))
		args = append(args, id)
	}
	rows, err := s.entClient.QueryContext(ctx, fmt.Sprintf(`
SELECT id, tags, total_quota, daily_quota, display_notes
FROM subscription_plans
WHERE id IN (%s)`, strings.Join(placeholders, ",")), args...)
	if err != nil {
		return nil
	}
	defer func() { _ = rows.Close() }()

	result := make(map[int64]PlanDisplayInfo, len(ids))
	for rows.Next() {
		var id int64
		var tags string
		var totalQuota sql.NullFloat64
		var dailyQuota sql.NullFloat64
		var displayNotes string
		if err := rows.Scan(&id, &tags, &totalQuota, &dailyQuota, &displayNotes); err != nil {
			continue
		}
		info := PlanDisplayInfo{Tags: tags, DisplayNotes: displayNotes}
		if totalQuota.Valid {
			info.TotalQuota = &totalQuota.Float64
		}
		if dailyQuota.Valid {
			info.DailyQuota = &dailyQuota.Float64
		}
		result[id] = info
	}
	return result
}

func (s *PaymentConfigService) ListPlans(ctx context.Context) ([]*dbent.SubscriptionPlan, error) {
	return s.entClient.SubscriptionPlan.Query().Order(subscriptionplan.BySortOrder()).All(ctx)
}

func (s *PaymentConfigService) ListPlansForSale(ctx context.Context) ([]*dbent.SubscriptionPlan, error) {
	return s.entClient.SubscriptionPlan.Query().Where(subscriptionplan.ForSaleEQ(true)).Order(subscriptionplan.BySortOrder()).All(ctx)
}

func (s *PaymentConfigService) CreatePlan(ctx context.Context, req CreatePlanRequest) (*dbent.SubscriptionPlan, error) {
	if err := validatePlanRequired(req.Name, req.GroupID, req.Price, req.ValidityDays, req.ValidityUnit, req.OriginalPrice); err != nil {
		return nil, err
	}
	if err := validatePlanDisplayFields(req.Tags, req.Features, req.TotalQuota, req.DailyQuota); err != nil {
		return nil, err
	}
	groupInfo, err := s.getActiveSubscriptionGroup(ctx, req.GroupID)
	if err != nil {
		return nil, err
	}
	derivedValidityUnit := derivePlanValidityUnitFromGroup(groupInfo)
	derivedQuota := derivePlanQuotaFromGroup(groupInfo, req.ValidityDays, derivedValidityUnit)
	b := s.entClient.SubscriptionPlan.Create().
		SetGroupID(req.GroupID).SetName(req.Name).SetDescription(req.Description).
		SetPrice(req.Price).SetValidityDays(req.ValidityDays).SetValidityUnit(derivedValidityUnit).
		SetFeatures(normalizeProductLines(req.Features)).SetProductName(req.ProductName).
		SetForSale(req.ForSale).SetSortOrder(req.SortOrder)
	if req.OriginalPrice != nil {
		b.SetOriginalPrice(*req.OriginalPrice)
	}
	plan, err := b.Save(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.updatePlanDisplayFields(ctx, plan.ID, &PlanDisplayInfo{
		Tags:         normalizeProductLines(req.Tags),
		TotalQuota:   derivedQuota.TotalQuota,
		DailyQuota:   derivedQuota.DailyQuota,
		DisplayNotes: strings.TrimSpace(req.DisplayNotes),
	}); err != nil {
		_ = s.entClient.SubscriptionPlan.DeleteOneID(plan.ID).Exec(ctx)
		return nil, err
	}
	return s.GetPlan(ctx, plan.ID)
}

// UpdatePlan updates a subscription plan by ID (patch semantics).
// NOTE: This function exceeds 30 lines due to per-field nil-check patch update boilerplate
// plus a validation guard for non-nil fields.
func (s *PaymentConfigService) UpdatePlan(ctx context.Context, id int64, req UpdatePlanRequest) (*dbent.SubscriptionPlan, error) {
	if err := validatePlanPatch(req); err != nil {
		return nil, err
	}
	currentPlan, err := s.entClient.SubscriptionPlan.Get(ctx, id)
	if err != nil {
		return nil, infraerrors.NotFound("PLAN_NOT_FOUND", "subscription plan not found")
	}
	nextGroupID := currentPlan.GroupID
	isFullPlanSave := req.GroupID != nil && req.ValidityDays != nil && req.ValidityUnit != nil
	if req.GroupID != nil {
		nextGroupID = *req.GroupID
	}
	nextValidityDays := currentPlan.ValidityDays
	if req.ValidityDays != nil {
		nextValidityDays = *req.ValidityDays
	}
	shouldDeriveQuota := isFullPlanSave || req.GroupID != nil || req.ValidityDays != nil || req.ValidityUnit != nil || req.TotalQuota != nil || req.DailyQuota != nil
	var derivedQuota planDerivedQuota
	var derivedValidityUnit string
	if shouldDeriveQuota {
		groupInfo, err := s.getActiveSubscriptionGroup(ctx, nextGroupID)
		if err != nil {
			return nil, err
		}
		derivedValidityUnit = derivePlanValidityUnitFromGroup(groupInfo)
		derivedQuota = derivePlanQuotaFromGroup(groupInfo, nextValidityDays, derivedValidityUnit)
	}
	u := s.entClient.SubscriptionPlan.UpdateOneID(id)
	if req.GroupID != nil {
		u.SetGroupID(*req.GroupID)
	}
	if req.Name != nil {
		u.SetName(*req.Name)
	}
	if req.Description != nil {
		u.SetDescription(*req.Description)
	}
	if req.Price != nil {
		u.SetPrice(*req.Price)
	}
	if req.OriginalPrice != nil {
		u.SetOriginalPrice(*req.OriginalPrice)
	}
	if req.ValidityDays != nil {
		u.SetValidityDays(*req.ValidityDays)
	}
	if shouldDeriveQuota {
		u.SetValidityUnit(derivedValidityUnit)
	}
	if req.Features != nil {
		features := normalizeProductLines(*req.Features)
		u.SetFeatures(features)
	}
	if req.ProductName != nil {
		u.SetProductName(*req.ProductName)
	}
	if req.ForSale != nil {
		u.SetForSale(*req.ForSale)
	}
	if req.SortOrder != nil {
		u.SetSortOrder(*req.SortOrder)
	}
	plan, err := u.Save(ctx)
	if err != nil {
		return nil, err
	}
	if req.Tags != nil || req.TotalQuota != nil || req.DailyQuota != nil || req.DisplayNotes != nil || shouldDeriveQuota {
		current := s.GetPlanDisplayInfoMap(ctx, []*dbent.SubscriptionPlan{plan})[plan.ID]
		next := current
		if req.Tags != nil {
			next.Tags = normalizeProductLines(*req.Tags)
		}
		if shouldDeriveQuota {
			next.TotalQuota = derivedQuota.TotalQuota
			next.DailyQuota = derivedQuota.DailyQuota
		}
		if req.DisplayNotes != nil {
			next.DisplayNotes = strings.TrimSpace(*req.DisplayNotes)
		}
		if err := s.updatePlanDisplayFields(ctx, plan.ID, &next); err != nil {
			return nil, err
		}
	}
	return s.GetPlan(ctx, id)
}

func (s *PaymentConfigService) getActiveSubscriptionGroup(ctx context.Context, groupID int64) (*dbent.Group, error) {
	groupInfo, err := s.entClient.Group.Get(ctx, groupID)
	if err != nil || groupInfo.Status != StatusActive {
		return nil, infraerrors.NotFound("GROUP_NOT_FOUND", "subscription group is no longer available")
	}
	if groupInfo.SubscriptionType != SubscriptionTypeSubscription {
		return nil, infraerrors.BadRequest("GROUP_TYPE_MISMATCH", "group is not a subscription type")
	}
	return groupInfo, nil
}

func (s *PaymentConfigService) updatePlanDisplayFields(ctx context.Context, planID int64, info *PlanDisplayInfo) error {
	if info == nil {
		return nil
	}
	var totalQuota any
	if info.TotalQuota != nil {
		totalQuota = *info.TotalQuota
	}
	var dailyQuota any
	if info.DailyQuota != nil {
		dailyQuota = *info.DailyQuota
	}
	_, err := s.entClient.ExecContext(ctx, `
UPDATE subscription_plans
SET tags = $2, total_quota = $3, daily_quota = $4, display_notes = $5, updated_at = $6
WHERE id = $1`, planID, normalizeProductLines(info.Tags), totalQuota, dailyQuota, strings.TrimSpace(info.DisplayNotes), time.Now())
	if err != nil {
		if isMissingPlanDisplayColumnError(err) {
			return nil
		}
		return fmt.Errorf("update plan display fields: %w", err)
	}
	return nil
}

func isMissingPlanDisplayColumnError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "no such column") ||
		strings.Contains(msg, "does not exist") ||
		strings.Contains(msg, "unknown column")
}

func (s *PaymentConfigService) DeletePlan(ctx context.Context, id int64) error {
	count, err := s.countPendingOrdersByPlan(ctx, id)
	if err != nil {
		return fmt.Errorf("check pending orders: %w", err)
	}
	if count > 0 {
		return infraerrors.Conflict("PENDING_ORDERS",
			fmt.Sprintf("this plan has %d in-progress orders and cannot be deleted — wait for orders to complete first", count))
	}
	return s.entClient.SubscriptionPlan.DeleteOneID(id).Exec(ctx)
}

// GetPlan returns a subscription plan by ID.
func (s *PaymentConfigService) GetPlan(ctx context.Context, id int64) (*dbent.SubscriptionPlan, error) {
	plan, err := s.entClient.SubscriptionPlan.Get(ctx, id)
	if err != nil {
		return nil, infraerrors.NotFound("PLAN_NOT_FOUND", "subscription plan not found")
	}
	return plan, nil
}
