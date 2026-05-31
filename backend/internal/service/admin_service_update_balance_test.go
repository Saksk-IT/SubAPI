//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type balanceUserRepoStub struct {
	*userRepoStub
	updateErr error
	updated   []*User
	users     map[int64]*User
}

func (s *balanceUserRepoStub) GetByID(ctx context.Context, id int64) (*User, error) {
	if s.users != nil {
		user, ok := s.users[id]
		if !ok {
			return nil, ErrUserNotFound
		}
		return user, nil
	}
	return s.userRepoStub.GetByID(ctx, id)
}

func (s *balanceUserRepoStub) Update(ctx context.Context, user *User) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	if user == nil {
		return nil
	}
	clone := *user
	s.updated = append(s.updated, &clone)
	if s.userRepoStub != nil {
		s.userRepoStub.user = &clone
	}
	if s.users != nil {
		s.users[user.ID] = &clone
	}
	return nil
}

type balanceRedeemRepoStub struct {
	*redeemRepoStub
	created []*RedeemCode
}

func (s *balanceRedeemRepoStub) Create(ctx context.Context, code *RedeemCode) error {
	if code == nil {
		return nil
	}
	clone := *code
	s.created = append(s.created, &clone)
	return nil
}

type authCacheInvalidatorStub struct {
	userIDs  []int64
	groupIDs []int64
	keys     []string
}

func (s *authCacheInvalidatorStub) InvalidateAuthCacheByKey(ctx context.Context, key string) {
	s.keys = append(s.keys, key)
}

func (s *authCacheInvalidatorStub) InvalidateAuthCacheByUserID(ctx context.Context, userID int64) {
	s.userIDs = append(s.userIDs, userID)
}

func (s *authCacheInvalidatorStub) InvalidateAuthCacheByGroupID(ctx context.Context, groupID int64) {
	s.groupIDs = append(s.groupIDs, groupID)
}

func TestAdminService_UpdateUserBalance_InvalidatesAuthCache(t *testing.T) {
	baseRepo := &userRepoStub{user: &User{ID: 7, Balance: 10}}
	repo := &balanceUserRepoStub{userRepoStub: baseRepo}
	redeemRepo := &balanceRedeemRepoStub{redeemRepoStub: &redeemRepoStub{}}
	invalidator := &authCacheInvalidatorStub{}
	svc := &adminServiceImpl{
		userRepo:             repo,
		redeemCodeRepo:       redeemRepo,
		authCacheInvalidator: invalidator,
	}

	_, err := svc.UpdateUserBalance(context.Background(), 7, 5, "add", "")
	require.NoError(t, err)
	require.Equal(t, []int64{7}, invalidator.userIDs)
	require.Len(t, redeemRepo.created, 1)
}

func TestAdminService_UpdateUserBalance_NoChangeNoInvalidate(t *testing.T) {
	baseRepo := &userRepoStub{user: &User{ID: 7, Balance: 10}}
	repo := &balanceUserRepoStub{userRepoStub: baseRepo}
	redeemRepo := &balanceRedeemRepoStub{redeemRepoStub: &redeemRepoStub{}}
	invalidator := &authCacheInvalidatorStub{}
	svc := &adminServiceImpl{
		userRepo:             repo,
		redeemCodeRepo:       redeemRepo,
		authCacheInvalidator: invalidator,
	}

	_, err := svc.UpdateUserBalance(context.Background(), 7, 10, "set", "")
	require.NoError(t, err)
	require.Empty(t, invalidator.userIDs)
	require.Empty(t, redeemRepo.created)
}

type batchAssignSubscriptionAssignerStub struct {
	extended map[int64]bool
	calls    []AssignSubscriptionInput
}

func (s *batchAssignSubscriptionAssignerStub) AssignOrExtendSubscription(ctx context.Context, input *AssignSubscriptionInput) (*UserSubscription, bool, error) {
	if input == nil {
		return nil, false, ErrSubscriptionNilInput
	}
	copied := *input
	s.calls = append(s.calls, copied)
	extended := s.extended[input.UserID]
	return &UserSubscription{
		ID:        input.UserID + 1000,
		UserID:    input.UserID,
		GroupID:   input.GroupID,
		Status:    SubscriptionStatusActive,
		StartsAt:  time.Now(),
		ExpiresAt: time.Now().AddDate(0, 0, input.ValidityDays),
	}, extended, nil
}

func TestAdminService_BatchAssignUsers_AddsBalanceForAllUsers(t *testing.T) {
	users := []User{{ID: 1, Balance: 10}, {ID: 2, Balance: 20}}
	userMap := map[int64]*User{
		1: &users[0],
		2: &users[1],
	}
	repo := &balanceUserRepoStub{
		userRepoStub: &userRepoStub{listUsers: users},
		users:        userMap,
	}
	redeemRepo := &balanceRedeemRepoStub{redeemRepoStub: &redeemRepoStub{}}
	svc := &adminServiceImpl{
		userRepo:       repo,
		redeemCodeRepo: redeemRepo,
	}

	result, err := svc.BatchAssignUsers(context.Background(), &BatchAssignUsersInput{
		Target: BatchAssignUserTarget{All: true},
		Balance: &BatchAssignBalanceInput{
			Operation: "add",
			Amount:    5,
			Notes:     "campaign",
		},
	})

	require.NoError(t, err)
	require.Equal(t, 2, result.TargetCount)
	require.Equal(t, 2, result.SuccessCount)
	require.Equal(t, 2, result.BalanceAffectedCount)
	require.Equal(t, 15.0, repo.users[1].Balance)
	require.Equal(t, 25.0, repo.users[2].Balance)
	require.Len(t, redeemRepo.created, 2)
}

func TestAdminService_BatchAssignUsers_AdjustsBalanceByRules(t *testing.T) {
	users := []User{{ID: 1, Balance: 50}, {ID: 2, Balance: 100}, {ID: 3, Balance: -5}}
	repo := &balanceUserRepoStub{
		userRepoStub: &userRepoStub{},
		users: map[int64]*User{
			1: &users[0],
			2: &users[1],
			3: &users[2],
		},
	}
	redeemRepo := &balanceRedeemRepoStub{redeemRepoStub: &redeemRepoStub{}}
	svc := &adminServiceImpl{
		userRepo:       repo,
		redeemCodeRepo: redeemRepo,
	}

	result, err := svc.BatchAssignUsers(context.Background(), &BatchAssignUsersInput{
		Target: BatchAssignUserTarget{UserIDs: []int64{1, 2, 3}},
		Balance: &BatchAssignBalanceInput{
			Operation: "rule",
			Rules: []BatchAssignBalanceRuleInput{
				{MinBalance: 0, MaxBalance: 100, Multiplier: 1.5},
				{MinBalance: 100, MaxBalance: 200, Multiplier: 1.2},
			},
			Notes: "tiered adjustment",
		},
	})

	require.NoError(t, err)
	require.Equal(t, 3, result.TargetCount)
	require.Equal(t, 3, result.SuccessCount)
	require.Equal(t, 2, result.BalanceAffectedCount)
	require.Equal(t, 75.0, repo.users[1].Balance)
	require.Equal(t, 120.0, repo.users[2].Balance)
	require.Equal(t, -5.0, repo.users[3].Balance)
	require.Len(t, redeemRepo.created, 2)
	require.Equal(t, 25.0, redeemRepo.created[0].Value)
	require.Equal(t, 20.0, redeemRepo.created[1].Value)
}

func TestAdminService_BatchAssignUsers_RuleAdjustmentRejectsUnmatchedWithoutWrites(t *testing.T) {
	users := []User{{ID: 1, Balance: 50}, {ID: 2, Balance: 250}}
	repo := &balanceUserRepoStub{
		userRepoStub: &userRepoStub{},
		users: map[int64]*User{
			1: &users[0],
			2: &users[1],
		},
	}
	redeemRepo := &balanceRedeemRepoStub{redeemRepoStub: &redeemRepoStub{}}
	svc := &adminServiceImpl{
		userRepo:       repo,
		redeemCodeRepo: redeemRepo,
	}

	result, err := svc.BatchAssignUsers(context.Background(), &BatchAssignUsersInput{
		Target: BatchAssignUserTarget{UserIDs: []int64{1, 2}},
		Balance: &BatchAssignBalanceInput{
			Operation: "rule",
			Rules: []BatchAssignBalanceRuleInput{
				{MinBalance: 0, MaxBalance: 100, Multiplier: 1.5},
			},
		},
	})

	require.Error(t, err)
	require.Nil(t, result)
	require.Empty(t, repo.updated)
	require.Empty(t, redeemRepo.created)
	require.Equal(t, 50.0, repo.users[1].Balance)
	require.Equal(t, 250.0, repo.users[2].Balance)
}

func TestValidateBatchAssignBalanceInput_Rules(t *testing.T) {
	require.NoError(t, validateBatchAssignBalanceInput(&BatchAssignBalanceInput{
		Operation: "rule",
		Rules: []BatchAssignBalanceRuleInput{
			{MinBalance: 100, MaxBalance: 200, Multiplier: 1.2},
			{MinBalance: 0, MaxBalance: 100, Multiplier: 1.5},
		},
	}))

	cases := []BatchAssignBalanceInput{
		{Operation: "rule"},
		{Operation: "rule", Rules: []BatchAssignBalanceRuleInput{{MinBalance: 0, MaxBalance: 100, Multiplier: 0}}},
		{Operation: "rule", Rules: []BatchAssignBalanceRuleInput{{MinBalance: 0, MaxBalance: 100, Multiplier: -1}}},
		{Operation: "rule", Rules: []BatchAssignBalanceRuleInput{{MinBalance: 100, MaxBalance: 100, Multiplier: 1}}},
		{Operation: "rule", Rules: []BatchAssignBalanceRuleInput{{MinBalance: 200, MaxBalance: 100, Multiplier: 1}}},
		{Operation: "rule", Rules: []BatchAssignBalanceRuleInput{{MinBalance: 0, MaxBalance: 100, Multiplier: 1}, {MinBalance: 50, MaxBalance: 150, Multiplier: 1}}},
		{Operation: "rule", Rules: []BatchAssignBalanceRuleInput{{MinBalance: 0, MaxBalance: 100, Multiplier: 1}, {MinBalance: 0, MaxBalance: 100, Multiplier: 1}}},
	}
	for _, tc := range cases {
		input := tc
		require.Error(t, validateBatchAssignBalanceInput(&input))
	}
}

func TestAdminService_BatchAssignUsers_SubtractBalanceReportsPartialFailure(t *testing.T) {
	users := []User{{ID: 1, Balance: 10}, {ID: 2, Balance: 2}}
	repo := &balanceUserRepoStub{
		userRepoStub: &userRepoStub{},
		users: map[int64]*User{
			1: &users[0],
			2: &users[1],
		},
	}
	redeemRepo := &balanceRedeemRepoStub{redeemRepoStub: &redeemRepoStub{}}
	svc := &adminServiceImpl{
		userRepo:       repo,
		redeemCodeRepo: redeemRepo,
	}

	result, err := svc.BatchAssignUsers(context.Background(), &BatchAssignUsersInput{
		Target: BatchAssignUserTarget{UserIDs: []int64{1, 2}},
		Balance: &BatchAssignBalanceInput{
			Operation: "subtract",
			Amount:    5,
		},
	})

	require.NoError(t, err)
	require.Equal(t, 2, result.TargetCount)
	require.Equal(t, 1, result.SuccessCount)
	require.Equal(t, 1, result.FailedCount)
	require.Equal(t, 1, result.BalanceAffectedCount)
	require.Len(t, result.Errors, 1)
	require.Equal(t, 5.0, repo.users[1].Balance)
	require.Equal(t, 2.0, repo.users[2].Balance)
}

func TestAdminService_BatchAssignUsers_AssignsAndExtendsSubscriptions(t *testing.T) {
	assigner := &batchAssignSubscriptionAssignerStub{
		extended: map[int64]bool{2: true},
	}
	svc := &adminServiceImpl{
		userRepo:           &userRepoStub{},
		defaultSubAssigner: assigner,
	}

	result, err := svc.BatchAssignUsers(context.Background(), &BatchAssignUsersInput{
		Target: BatchAssignUserTarget{UserIDs: []int64{1, 2}},
		Subscription: &BatchAssignSubscriptionInput{
			GroupID:      9,
			ValidityDays: 30,
			AssignedBy:   7,
			Notes:        "manual grant",
		},
	})

	require.NoError(t, err)
	require.Equal(t, 2, result.SuccessCount)
	require.Equal(t, 1, result.SubscriptionAssigned)
	require.Equal(t, 1, result.SubscriptionExtended)
	require.Len(t, assigner.calls, 2)
	require.Equal(t, int64(7), assigner.calls[0].AssignedBy)
}
