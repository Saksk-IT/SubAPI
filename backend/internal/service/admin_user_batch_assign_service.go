package service

import (
	"context"
	"fmt"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

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
	if input.Amount <= 0 {
		return infraerrors.BadRequest("INVALID_BATCH_BALANCE_AMOUNT", "balance amount must be greater than 0")
	}
	switch input.Operation {
	case "add", "subtract":
		return nil
	default:
		return infraerrors.BadRequest("INVALID_BATCH_BALANCE_OPERATION", "balance operation must be add or subtract")
	}
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

func (s *adminServiceImpl) applyBatchAssignToUser(ctx context.Context, userID int64, input *BatchAssignUsersInput) (bool, bool, error) {
	balanceChanged := false
	if input.Balance != nil {
		user, err := s.UpdateUserBalance(ctx, userID, input.Balance.Amount, input.Balance.Operation, input.Balance.Notes)
		if err != nil {
			return false, false, err
		}
		balanceChanged = user != nil
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
