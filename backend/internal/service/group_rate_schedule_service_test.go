package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type groupRateScheduleSettingRepoStub struct {
	value string
}

func (r *groupRateScheduleSettingRepoStub) Get(context.Context, string) (*Setting, error) {
	return nil, errors.New("unexpected Get")
}

func (r *groupRateScheduleSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	if key != SettingKeyGroupRateScheduleSettings || r.value == "" {
		return "", ErrSettingNotFound
	}
	return r.value, nil
}

func (r *groupRateScheduleSettingRepoStub) Set(_ context.Context, key, value string) error {
	if key != SettingKeyGroupRateScheduleSettings {
		return errors.New("unexpected key")
	}
	r.value = value
	return nil
}

func (r *groupRateScheduleSettingRepoStub) GetMultiple(context.Context, []string) (map[string]string, error) {
	return nil, errors.New("unexpected GetMultiple")
}

func (r *groupRateScheduleSettingRepoStub) SetMultiple(context.Context, map[string]string) error {
	return errors.New("unexpected SetMultiple")
}

func (r *groupRateScheduleSettingRepoStub) GetAll(context.Context) (map[string]string, error) {
	return nil, errors.New("unexpected GetAll")
}

func (r *groupRateScheduleSettingRepoStub) Delete(context.Context, string) error {
	return errors.New("unexpected Delete")
}

type groupRateScheduleGroupRepoStub struct {
	groups  []Group
	updates []GroupRateMultiplierUpdate
}

func (r *groupRateScheduleGroupRepoStub) Create(context.Context, *Group) error { return nil }
func (r *groupRateScheduleGroupRepoStub) GetByID(context.Context, int64) (*Group, error) {
	return nil, ErrGroupNotFound
}
func (r *groupRateScheduleGroupRepoStub) GetByIDLite(context.Context, int64) (*Group, error) {
	return nil, ErrGroupNotFound
}
func (r *groupRateScheduleGroupRepoStub) Update(context.Context, *Group) error { return nil }
func (r *groupRateScheduleGroupRepoStub) Delete(context.Context, int64) error  { return nil }
func (r *groupRateScheduleGroupRepoStub) DeleteCascade(context.Context, int64) ([]int64, error) {
	return nil, nil
}
func (r *groupRateScheduleGroupRepoStub) List(context.Context, pagination.PaginationParams) ([]Group, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (r *groupRateScheduleGroupRepoStub) ListWithFilters(_ context.Context, params pagination.PaginationParams, _, _, _ string, _ *bool) ([]Group, *pagination.PaginationResult, error) {
	start := params.Offset()
	if start >= len(r.groups) {
		return []Group{}, &pagination.PaginationResult{Total: int64(len(r.groups)), Page: params.Page, PageSize: params.Limit(), Pages: 1}, nil
	}
	end := start + params.Limit()
	if end > len(r.groups) {
		end = len(r.groups)
	}
	return r.groups[start:end], &pagination.PaginationResult{Total: int64(len(r.groups)), Page: params.Page, PageSize: params.Limit(), Pages: 1}, nil
}
func (r *groupRateScheduleGroupRepoStub) ListActive(context.Context) ([]Group, error) {
	return r.groups, nil
}
func (r *groupRateScheduleGroupRepoStub) ListActiveByPlatform(context.Context, string) ([]Group, error) {
	return r.groups, nil
}
func (r *groupRateScheduleGroupRepoStub) ExistsByName(context.Context, string) (bool, error) {
	return false, nil
}
func (r *groupRateScheduleGroupRepoStub) GetAccountCount(context.Context, int64) (int64, int64, error) {
	return 0, 0, nil
}
func (r *groupRateScheduleGroupRepoStub) DeleteAccountGroupsByGroupID(context.Context, int64) (int64, error) {
	return 0, nil
}
func (r *groupRateScheduleGroupRepoStub) GetAccountIDsByGroupIDs(context.Context, []int64) ([]int64, error) {
	return nil, nil
}
func (r *groupRateScheduleGroupRepoStub) BindAccountsToGroup(context.Context, int64, []int64) error {
	return nil
}
func (r *groupRateScheduleGroupRepoStub) UpdateSortOrders(context.Context, []GroupSortOrderUpdate) error {
	return nil
}
func (r *groupRateScheduleGroupRepoStub) UpdateRateMultipliers(_ context.Context, updates []GroupRateMultiplierUpdate) error {
	r.updates = append(r.updates, updates...)
	for _, update := range updates {
		for i := range r.groups {
			if r.groups[i].ID == update.ID {
				r.groups[i].RateMultiplier = update.RateMultiplier
			}
		}
	}
	return nil
}

type groupRateScheduleInvalidatorStub struct {
	groupIDs []int64
}

func (i *groupRateScheduleInvalidatorStub) InvalidateAuthCacheByKey(context.Context, string)   {}
func (i *groupRateScheduleInvalidatorStub) InvalidateAuthCacheByUserID(context.Context, int64) {}
func (i *groupRateScheduleInvalidatorStub) InvalidateAuthCacheByGroupID(_ context.Context, groupID int64) {
	i.groupIDs = append(i.groupIDs, groupID)
}

func TestGroupRateScheduleService_ApplyAndRestore(t *testing.T) {
	repo := &groupRateScheduleSettingRepoStub{}
	groupRepo := &groupRateScheduleGroupRepoStub{groups: []Group{
		{ID: 1, RateMultiplier: 1.0},
		{ID: 2, RateMultiplier: 1.5},
	}}
	invalidator := &groupRateScheduleInvalidatorStub{}
	svc := NewGroupRateScheduleService(repo, groupRepo, invalidator)
	svc.now = func() time.Time {
		return time.Date(2026, 5, 24, 10, 30, 0, 0, time.UTC)
	}

	settings, err := svc.UpdateSettings(context.Background(), &GroupRateScheduleSettings{
		Enabled:   true,
		StartTime: "10:00",
		EndTime:   "11:00",
		Percent:   85,
	})
	require.NoError(t, err)
	require.True(t, settings.Active)
	require.Equal(t, 0.85, groupRepo.groups[0].RateMultiplier)
	require.Equal(t, 0.85, groupRepo.groups[1].RateMultiplier)
	require.Equal(t, map[string]float64{"1": 1.0, "2": 1.5}, settings.OriginalRates)
	require.ElementsMatch(t, []int64{1, 2}, invalidator.groupIDs)

	svc.now = func() time.Time {
		return time.Date(2026, 5, 24, 11, 0, 0, 0, time.UTC)
	}
	err = svc.evaluate(context.Background())
	require.NoError(t, err)
	settings, err = svc.GetSettings(context.Background())
	require.NoError(t, err)
	require.False(t, settings.Active)
	require.Empty(t, settings.OriginalRates)
	require.Equal(t, 1.0, groupRepo.groups[0].RateMultiplier)
	require.Equal(t, 1.5, groupRepo.groups[1].RateMultiplier)
}

func TestGroupRateScheduleService_ApplyRefreshesActiveSchedule(t *testing.T) {
	repo := &groupRateScheduleSettingRepoStub{}
	groupRepo := &groupRateScheduleGroupRepoStub{groups: []Group{
		{ID: 1, RateMultiplier: 1.0},
		{ID: 2, RateMultiplier: 1.5},
	}}
	svc := NewGroupRateScheduleService(repo, groupRepo, nil)
	svc.now = func() time.Time {
		return time.Date(2026, 5, 24, 10, 30, 0, 0, time.UTC)
	}

	settings, err := svc.UpdateSettings(context.Background(), &GroupRateScheduleSettings{
		Enabled:   true,
		StartTime: "10:00",
		EndTime:   "11:00",
		Percent:   85,
	})
	require.NoError(t, err)
	require.True(t, settings.Active)

	groupRepo.groups = append(groupRepo.groups, Group{ID: 3, RateMultiplier: 2.0})
	settings, err = svc.UpdateSettings(context.Background(), &GroupRateScheduleSettings{
		Enabled:   true,
		StartTime: "10:00",
		EndTime:   "11:00",
		Percent:   90,
	})
	require.NoError(t, err)
	require.True(t, settings.Active)
	require.Equal(t, 0.9, groupRepo.groups[0].RateMultiplier)
	require.Equal(t, 0.9, groupRepo.groups[1].RateMultiplier)
	require.Equal(t, 0.9, groupRepo.groups[2].RateMultiplier)
	require.Equal(t, map[string]float64{"1": 1.0, "2": 1.5, "3": 2.0}, settings.OriginalRates)
}

func TestGroupRateScheduleSettings_InWindowAcrossMidnight(t *testing.T) {
	settings := &GroupRateScheduleSettings{
		Enabled:   true,
		StartTime: "23:00",
		EndTime:   "02:00",
		Percent:   85,
	}
	active, err := settings.inWindow(time.Date(2026, 5, 24, 23, 30, 0, 0, time.UTC))
	require.NoError(t, err)
	require.True(t, active)

	active, err = settings.inWindow(time.Date(2026, 5, 25, 1, 30, 0, 0, time.UTC))
	require.NoError(t, err)
	require.True(t, active)

	active, err = settings.inWindow(time.Date(2026, 5, 25, 2, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	require.False(t, active)
}

func TestNormalizeGroupRateScheduleSettings_RejectsInvalidPercent(t *testing.T) {
	_, err := normalizeGroupRateScheduleSettings(&GroupRateScheduleSettings{
		Enabled:   true,
		StartTime: "09:00",
		EndTime:   "10:00",
		Percent:   -1,
	})
	require.Error(t, err)

	_, err = normalizeGroupRateScheduleSettings(&GroupRateScheduleSettings{
		Enabled:   true,
		StartTime: "09:00",
		EndTime:   "10:00",
		Percent:   101,
	})
	require.Error(t, err)
}
