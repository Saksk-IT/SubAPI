package service

import (
	"context"
	"fmt"
	"math"
	"sort"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

const batchAssignBalanceOperationRule = "rule"

func (s *adminServiceImpl) BatchAssignUsers(ctx context.Context, input *BatchAssignUsersInput) (*BatchAssignUsersResult, error) {
	if input == nil {
		return nil, infraerrors.BadRequest("INVALID_BATCH_ASSIGN_INPUT", "batch assign input is required")
	}
	if input.Balance == nil && input.Subscription == nil {
		return nil, infraerrors.BadRequest("INVALID_BATCH_ASSIGN_ACTION", "at least one batch action is required")
	}
	if input.Balance != nil && input.Subscription != nil {
		return nil, infraerrors.BadRequest("INVALID_BATCH_ASSIGN_ACTION", "only one batch action can be applied at a time")
	}
	if input.Balance != nil {
		if err := validateBatchAssignBalanceInput(input.Balance); err != nil {
			return nil, err
		}
	}
	if input.Subscription != nil {
		if err := validateBatchAssignSubscriptionInput(input.Subscription, s.defaultSubAssigner); err != nil {
			return nil, err
		}
	}

	userIDs, err := s.resolveBatchAssignUserIDs(ctx, input.Target)
	if err != nil {
		return nil, err
	}
	if input.Balance != nil && input.Balance.Operation == batchAssignBalanceOperationRule {
		if err := s.ensureBatchAssignBalanceRuleCoverage(ctx, userIDs, input.Balance.Rules); err != nil {
			return nil, err
		}
	}

	result := &BatchAssignUsersResult{
		TargetCount: len(userIDs),
		Errors:      make([]string, 0),
	}
	for _, userID := range userIDs {
		changed, extended, err := s.applyBatchAssignToUser(ctx, userID, input)
		if err != nil {
			result.FailedCount++
			result.Errors = append(result.Errors, fmt.Sprintf("user %d: %v", userID, err))
			continue
		}
		result.SuccessCount++
		if input.Balance != nil && changed {
			result.BalanceAffectedCount++
		}
		if input.Subscription != nil {
			if extended {
				result.SubscriptionExtended++
			} else {
				result.SubscriptionAssigned++
			}
		}
	}

	return result, nil
}

func validateBatchAssignBalanceInput(input *BatchAssignBalanceInput) error {
	if input == nil {
		return nil
	}
	switch input.Operation {
	case "add", "subtract":
		if !isFiniteNumber(input.Amount) || input.Amount <= 0 {
			return infraerrors.BadRequest("INVALID_BATCH_BALANCE_AMOUNT", "balance amount must be greater than 0")
		}
		return nil
	case batchAssignBalanceOperationRule:
		return validateBatchAssignBalanceRules(input.Rules)
	default:
		return infraerrors.BadRequest("INVALID_BATCH_BALANCE_OPERATION", "balance operation must be add, subtract, or rule")
	}
}

func validateBatchAssignBalanceRules(rules []BatchAssignBalanceRuleInput) error {
	if len(rules) == 0 {
		return infraerrors.BadRequest("INVALID_BATCH_BALANCE_RULES", "at least one balance rule is required")
	}
	sorted := sortedBatchAssignBalanceRules(rules)
	for i, rule := range sorted {
		if !isFiniteNumber(rule.MinBalance) || !isFiniteNumber(rule.MaxBalance) || !isFiniteNumber(rule.Multiplier) {
			return infraerrors.BadRequest("INVALID_BATCH_BALANCE_RULES", "balance rule values must be finite numbers")
		}
		if rule.MinBalance < 0 {
			return infraerrors.BadRequest("INVALID_BATCH_BALANCE_RULES", "balance rule minimum must be greater than or equal to 0")
		}
		if rule.MaxBalance <= rule.MinBalance {
			return infraerrors.BadRequest("INVALID_BATCH_BALANCE_RULES", "balance rule maximum must be greater than minimum")
		}
		if rule.Multiplier <= 0 {
			return infraerrors.BadRequest("INVALID_BATCH_BALANCE_RULES", "balance rule multiplier must be greater than 0")
		}
		if i > 0 && rule.MinBalance < sorted[i-1].MaxBalance {
			return infraerrors.BadRequest("INVALID_BATCH_BALANCE_RULES", "balance rules must not overlap")
		}
	}
	return nil
}

func sortedBatchAssignBalanceRules(rules []BatchAssignBalanceRuleInput) []BatchAssignBalanceRuleInput {
	sorted := append([]BatchAssignBalanceRuleInput(nil), rules...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].MinBalance == sorted[j].MinBalance {
			return sorted[i].MaxBalance < sorted[j].MaxBalance
		}
		return sorted[i].MinBalance < sorted[j].MinBalance
	})
	return sorted
}

func isFiniteNumber(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func validateBatchAssignSubscriptionInput(input *BatchAssignSubscriptionInput, assigner DefaultSubscriptionAssigner) error {
	if input == nil {
		return nil
	}
	if assigner == nil {
		return infraerrors.InternalServer("SUBSCRIPTION_SERVICE_UNAVAILABLE", "subscription service is not configured")
	}
	if input.GroupID <= 0 {
		return infraerrors.BadRequest("INVALID_BATCH_SUBSCRIPTION_GROUP", "subscription group is required")
	}
	if input.ValidityDays <= 0 {
		return infraerrors.BadRequest("INVALID_BATCH_SUBSCRIPTION_DAYS", "subscription validity days must be greater than 0")
	}
	if input.ValidityDays > MaxValidityDays {
		return infraerrors.BadRequest("INVALID_BATCH_SUBSCRIPTION_DAYS", "subscription validity days exceeds the maximum")
	}
	return nil
}

func (s *adminServiceImpl) resolveBatchAssignUserIDs(ctx context.Context, target BatchAssignUserTarget) ([]int64, error) {
	if target.All {
		return s.listAllUserIDsForBatchAssign(ctx)
	}
	cleaned := dedupePositiveInt64s(target.UserIDs)
	if len(cleaned) == 0 {
		return nil, infraerrors.BadRequest("INVALID_BATCH_ASSIGN_TARGET", "user_ids is required unless all=true")
	}
	return cleaned, nil
}

func (s *adminServiceImpl) listAllUserIDsForBatchAssign(ctx context.Context) ([]int64, error) {
	const pageSize = 500
	userIDs := make([]int64, 0)
	for page := 1; ; page++ {
		includeSubscriptions := false
		users, result, err := s.userRepo.ListWithFilters(ctx, pagination.PaginationParams{
			Page:      page,
			PageSize:  pageSize,
			SortBy:    "id",
			SortOrder: "asc",
		}, UserListFilters{IncludeSubscriptions: &includeSubscriptions})
		if err != nil {
			return nil, err
		}
		for i := range users {
			userIDs = append(userIDs, users[i].ID)
		}
		total := int64(0)
		if result != nil {
			total = result.Total
		}
		if len(users) < pageSize || (total > 0 && int64(len(userIDs)) >= total) {
			break
		}
	}
	return userIDs, nil
}

func dedupePositiveInt64s(ids []int64) []int64 {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[int64]struct{}, len(ids))
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func (s *adminServiceImpl) ensureBatchAssignBalanceRuleCoverage(ctx context.Context, userIDs []int64, rules []BatchAssignBalanceRuleInput) error {
	for _, userID := range userIDs {
		user, err := s.userRepo.GetByID(ctx, userID)
		if err != nil {
			return err
		}
		if _, ok := findBatchAssignBalanceRule(user.Balance, rules); !ok {
			return infraerrors.BadRequest(
				"BATCH_BALANCE_RULE_UNMATCHED_USER",
				fmt.Sprintf("user %d balance %.2f does not match any balance rule", userID, user.Balance),
			)
		}
	}
	return nil
}

func findBatchAssignBalanceRule(balance float64, rules []BatchAssignBalanceRuleInput) (BatchAssignBalanceRuleInput, bool) {
	for _, rule := range rules {
		if balance >= rule.MinBalance && balance < rule.MaxBalance {
			return rule, true
		}
	}
	return BatchAssignBalanceRuleInput{}, false
}

func (s *adminServiceImpl) applyBatchAssignBalanceToUser(ctx context.Context, userID int64, input *BatchAssignBalanceInput) (bool, error) {
	if input.Operation != batchAssignBalanceOperationRule {
		user, err := s.UpdateUserBalance(ctx, userID, input.Amount, input.Operation, input.Notes)
		return user != nil, err
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return false, err
	}
	rule, ok := findBatchAssignBalanceRule(user.Balance, input.Rules)
	if !ok {
		return false, infraerrors.BadRequest(
			"BATCH_BALANCE_RULE_UNMATCHED_USER",
			fmt.Sprintf("user %d balance %.2f does not match any balance rule", userID, user.Balance),
		)
	}
	adjustedBalance := user.Balance * rule.Multiplier
	if !isFiniteNumber(adjustedBalance) {
		return false, infraerrors.BadRequest("INVALID_BATCH_BALANCE_RULES", "balance rule result must be a finite number")
	}
	updatedUser, err := s.UpdateUserBalance(ctx, userID, adjustedBalance, "set", input.Notes)
	return updatedUser != nil, err
}

func (s *adminServiceImpl) applyBatchAssignToUser(ctx context.Context, userID int64, input *BatchAssignUsersInput) (bool, bool, error) {
	balanceChanged := false
	if input.Balance != nil {
		changed, err := s.applyBatchAssignBalanceToUser(ctx, userID, input.Balance)
		if err != nil {
			return false, false, err
		}
		balanceChanged = changed
	}

	subscriptionExtended := false
	if input.Subscription != nil {
		_, extended, err := s.defaultSubAssigner.AssignOrExtendSubscription(ctx, &AssignSubscriptionInput{
			UserID:       userID,
			GroupID:      input.Subscription.GroupID,
			ValidityDays: input.Subscription.ValidityDays,
			AssignedBy:   input.Subscription.AssignedBy,
			Notes:        input.Subscription.Notes,
		})
		if err != nil {
			return balanceChanged, false, err
		}
		subscriptionExtended = extended
	}

	return balanceChanged, subscriptionExtended, nil
}
