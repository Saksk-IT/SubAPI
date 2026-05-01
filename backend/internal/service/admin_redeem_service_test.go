//go:build unit

package service

import (
	"context"
	"testing"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type adminRedeemRepoStub struct {
	created []RedeemCode
	stored  map[int64]*RedeemCode
}

func (s *adminRedeemRepoStub) Create(ctx context.Context, code *RedeemCode) error {
	if s.stored == nil {
		s.stored = map[int64]*RedeemCode{}
	}
	created := *code
	if created.ID == 0 {
		created.ID = int64(len(s.created) + 1)
	}
	code.ID = created.ID
	s.created = append(s.created, created)
	s.stored[created.ID] = &created
	return nil
}

func (s *adminRedeemRepoStub) CreateBatch(context.Context, []RedeemCode) error { panic("unexpected") }
func (s *adminRedeemRepoStub) GetByCode(context.Context, string) (*RedeemCode, error) {
	panic("unexpected")
}
func (s *adminRedeemRepoStub) Delete(context.Context, int64) error     { panic("unexpected") }
func (s *adminRedeemRepoStub) Use(context.Context, int64, int64) error { panic("unexpected") }
func (s *adminRedeemRepoStub) List(context.Context, pagination.PaginationParams) ([]RedeemCode, *pagination.PaginationResult, error) {
	panic("unexpected")
}
func (s *adminRedeemRepoStub) ListWithFilters(context.Context, pagination.PaginationParams, string, string, string) ([]RedeemCode, *pagination.PaginationResult, error) {
	panic("unexpected")
}
func (s *adminRedeemRepoStub) ListByUser(context.Context, int64, int) ([]RedeemCode, error) {
	panic("unexpected")
}
func (s *adminRedeemRepoStub) ListByUserPaginated(context.Context, int64, pagination.PaginationParams, string) ([]RedeemCode, *pagination.PaginationResult, error) {
	panic("unexpected")
}
func (s *adminRedeemRepoStub) SumPositiveBalanceByUser(context.Context, int64) (float64, error) {
	panic("unexpected")
}

func (s *adminRedeemRepoStub) GetByID(ctx context.Context, id int64) (*RedeemCode, error) {
	if s.stored != nil {
		if code, ok := s.stored[id]; ok {
			copy := *code
			return &copy, nil
		}
	}
	return nil, ErrRedeemCodeNotFound
}

func (s *adminRedeemRepoStub) Update(ctx context.Context, code *RedeemCode) error {
	if s.stored == nil {
		s.stored = map[int64]*RedeemCode{}
	}
	copy := *code
	s.stored[code.ID] = &copy
	return nil
}

type adminRedeemGroupRepoStub struct {
	groups map[int64]*Group
}

func (s *adminRedeemGroupRepoStub) GetByID(_ context.Context, id int64) (*Group, error) {
	if s.groups != nil {
		if group, ok := s.groups[id]; ok {
			copy := *group
			return &copy, nil
		}
	}
	return nil, ErrGroupNotFound
}

func (s *adminRedeemGroupRepoStub) Create(context.Context, *Group) error { panic("unexpected") }
func (s *adminRedeemGroupRepoStub) GetByIDLite(context.Context, int64) (*Group, error) {
	panic("unexpected")
}
func (s *adminRedeemGroupRepoStub) Update(context.Context, *Group) error { panic("unexpected") }
func (s *adminRedeemGroupRepoStub) Delete(context.Context, int64) error  { panic("unexpected") }
func (s *adminRedeemGroupRepoStub) DeleteCascade(context.Context, int64) ([]int64, error) {
	panic("unexpected")
}
func (s *adminRedeemGroupRepoStub) List(context.Context, pagination.PaginationParams) ([]Group, *pagination.PaginationResult, error) {
	panic("unexpected")
}
func (s *adminRedeemGroupRepoStub) ListWithFilters(context.Context, pagination.PaginationParams, string, string, string, *bool) ([]Group, *pagination.PaginationResult, error) {
	panic("unexpected")
}
func (s *adminRedeemGroupRepoStub) ListActive(context.Context) ([]Group, error) { panic("unexpected") }
func (s *adminRedeemGroupRepoStub) ListActiveByPlatform(context.Context, string) ([]Group, error) {
	panic("unexpected")
}
func (s *adminRedeemGroupRepoStub) ExistsByName(context.Context, string) (bool, error) {
	panic("unexpected")
}
func (s *adminRedeemGroupRepoStub) GetAccountCount(context.Context, int64) (int64, int64, error) {
	panic("unexpected")
}
func (s *adminRedeemGroupRepoStub) DeleteAccountGroupsByGroupID(context.Context, int64) (int64, error) {
	panic("unexpected")
}
func (s *adminRedeemGroupRepoStub) GetAccountIDsByGroupIDs(context.Context, []int64) ([]int64, error) {
	panic("unexpected")
}
func (s *adminRedeemGroupRepoStub) BindAccountsToGroup(context.Context, int64, []int64) error {
	panic("unexpected")
}
func (s *adminRedeemGroupRepoStub) UpdateSortOrders(context.Context, []GroupSortOrderUpdate) error {
	panic("unexpected")
}

func TestAdminService_CreateRedeemCodeBalance(t *testing.T) {
	repo := &adminRedeemRepoStub{}
	svc := &adminServiceImpl{redeemCodeRepo: repo}

	code, err := svc.CreateRedeemCode(context.Background(), &CreateRedeemCodeInput{
		Code:  " TEST-CODE ",
		Type:  RedeemTypeBalance,
		Value: 25,
	})

	require.NoError(t, err)
	require.Equal(t, "TEST-CODE", code.Code)
	require.Equal(t, RedeemTypeBalance, code.Type)
	require.Equal(t, StatusUnused, code.Status)
	require.Nil(t, code.GroupID)
	require.Zero(t, code.ValidityDays)
	require.Len(t, repo.created, 1)
}

func TestAdminService_GenerateRedeemCodesSubscriptionUsesGroupAndValidity(t *testing.T) {
	groupID := int64(9)
	repo := &adminRedeemRepoStub{}
	svc := &adminServiceImpl{
		redeemCodeRepo: repo,
		groupRepo: &adminRedeemGroupRepoStub{groups: map[int64]*Group{
			groupID: {ID: groupID, SubscriptionType: SubscriptionTypeSubscription},
		}},
	}

	codes, err := svc.GenerateRedeemCodes(context.Background(), &GenerateRedeemCodesInput{
		Count:        2,
		Type:         RedeemTypeSubscription,
		GroupID:      &groupID,
		ValidityDays: 45,
	})

	require.NoError(t, err)
	require.Len(t, codes, 2)
	require.Len(t, repo.created, 2)
	for _, code := range repo.created {
		require.Equal(t, RedeemTypeSubscription, code.Type)
		require.NotNil(t, code.GroupID)
		require.Equal(t, groupID, *code.GroupID)
		require.Equal(t, 45, code.ValidityDays)
	}
}

func TestAdminService_GenerateRedeemCodesRejectsStandardGroup(t *testing.T) {
	groupID := int64(3)
	svc := &adminServiceImpl{
		redeemCodeRepo: &adminRedeemRepoStub{},
		groupRepo: &adminRedeemGroupRepoStub{groups: map[int64]*Group{
			groupID: {ID: groupID, SubscriptionType: SubscriptionTypeStandard},
		}},
	}

	_, err := svc.GenerateRedeemCodes(context.Background(), &GenerateRedeemCodesInput{
		Count:   1,
		Type:    RedeemTypeSubscription,
		GroupID: &groupID,
	})

	require.Error(t, err)
	require.Equal(t, "INVALID_REDEEM_CODE_GROUP", infraerrors.Reason(err))
}

func TestAdminService_UpdateRedeemCodeKeepsImmutableCopy(t *testing.T) {
	groupID := int64(4)
	repo := &adminRedeemRepoStub{stored: map[int64]*RedeemCode{
		1: {ID: 1, Code: "OLD", Type: RedeemTypeBalance, Value: 10, Status: StatusUnused},
	}}
	svc := &adminServiceImpl{
		redeemCodeRepo: repo,
		groupRepo: &adminRedeemGroupRepoStub{groups: map[int64]*Group{
			groupID: {ID: groupID, SubscriptionType: SubscriptionTypeSubscription},
		}},
	}
	typeValue := RedeemTypeSubscription
	validity := 60

	updated, err := svc.UpdateRedeemCode(context.Background(), 1, &UpdateRedeemCodeInput{
		Type:         &typeValue,
		GroupID:      &groupID,
		ValidityDays: &validity,
	})

	require.NoError(t, err)
	require.Equal(t, RedeemTypeSubscription, updated.Type)
	require.NotNil(t, updated.GroupID)
	require.Equal(t, groupID, *updated.GroupID)
	require.Equal(t, 60, updated.ValidityDays)
}
