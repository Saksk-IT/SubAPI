package service

import (
	"context"
	"sort"
	"strings"
	"testing"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

func TestFirstRechargeActivityServiceStatusScopes(t *testing.T) {
	eligibleSince := time.Date(2026, 1, 10, 8, 0, 0, 0, time.UTC)
	oldUser := &User{ID: 1, Email: "old@example.test", Username: "old", CreatedAt: eligibleSince.Add(-time.Hour)}
	newUser := &User{ID: 2, Email: "new@example.test", Username: "new", CreatedAt: eligibleSince.Add(time.Hour)}

	tests := []struct {
		name       string
		scope      string
		userID     int64
		specified  map[int64]bool
		wantElig   bool
		wantOffers int
	}{
		{
			name:       "all users includes old users",
			scope:      FirstRechargeEligibilityAllUsers,
			userID:     oldUser.ID,
			wantElig:   true,
			wantOffers: 1,
		},
		{
			name:     "new users mode excludes users before eligible since",
			scope:    FirstRechargeEligibilityNewUsersAfterEnabled,
			userID:   oldUser.ID,
			wantElig: false,
		},
		{
			name:       "new users mode includes users after eligible since",
			scope:      FirstRechargeEligibilityNewUsersAfterEnabled,
			userID:     newUser.ID,
			wantElig:   true,
			wantOffers: 1,
		},
		{
			name:      "specified users excludes missing user",
			scope:     FirstRechargeEligibilitySpecifiedUsers,
			userID:    newUser.ID,
			specified: map[int64]bool{},
			wantElig:  false,
		},
		{
			name:       "specified users includes listed user",
			scope:      FirstRechargeEligibilitySpecifiedUsers,
			userID:     newUser.ID,
			specified:  map[int64]bool{newUser.ID: true},
			wantElig:   true,
			wantOffers: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFirstRechargeMemoryRepo()
			repo.config = FirstRechargeConfig{
				Enabled:          true,
				EligibilityScope: tt.scope,
				EligibleSince:    &eligibleSince,
			}
			repo.offers = []FirstRechargeOffer{{
				ID:      10,
				Name:    "starter",
				Price:   9.9,
				Amount:  19.9,
				Enabled: true,
			}}
			repo.specified = tt.specified
			svc := NewFirstRechargeActivityService(repo, &firstRechargeUserRepoFake{
				users: map[int64]*User{oldUser.ID: oldUser, newUser.ID: newUser},
			})

			status, err := svc.GetStatus(context.Background(), tt.userID)
			require.NoError(t, err)
			require.Equal(t, tt.wantElig, status.Eligible)
			require.Equal(t, tt.wantOffers, len(status.Offers))
		})
	}
}

func TestFirstRechargeActivityServiceDismissPopupAndCompletion(t *testing.T) {
	repo := newFirstRechargeMemoryRepo()
	repo.config.Enabled = true
	repo.config.EligibilityScope = FirstRechargeEligibilityAllUsers
	repo.offers = []FirstRechargeOffer{{ID: 7, Name: "pack", Price: 10, Amount: 20, Enabled: true}}
	userRepo := &firstRechargeUserRepoFake{
		users: map[int64]*User{
			3: {ID: 3, Email: "user@example.test", Username: "user", CreatedAt: time.Now().Add(-time.Hour)},
		},
	}
	svc := NewFirstRechargeActivityService(repo, userRepo)

	require.NoError(t, svc.DismissPopup(context.Background(), 3))
	status, err := svc.GetStatus(context.Background(), 3)
	require.NoError(t, err)
	require.True(t, status.PopupDismissed)
	require.True(t, status.Eligible)

	require.NoError(t, svc.MarkCompleted(context.Background(), 3, 99))
	status, err = svc.GetStatus(context.Background(), 3)
	require.NoError(t, err)
	require.True(t, status.Completed)
	require.False(t, status.Eligible)
	require.Empty(t, status.Offers)
	require.NotNil(t, status.CompletedAt)

	_, err = svc.PrepareOrder(context.Background(), 3, 7)
	require.Equal(t, "FIRST_RECHARGE_COMPLETED", infraerrors.Reason(err))
}

func TestFirstRechargeActivityServiceUpdateAdminConfig(t *testing.T) {
	repo := newFirstRechargeMemoryRepo()
	svc := NewFirstRechargeActivityService(repo, &firstRechargeUserRepoFake{})

	config, err := svc.UpdateAdminConfig(context.Background(), UpdateFirstRechargeConfigInput{
		Enabled:          true,
		EligibilityScope: FirstRechargeEligibilityNewUsersAfterEnabled,
		Offers: []FirstRechargeOfferInput{{
			Name:      "  新人首充  ",
			Price:     9.999,
			Amount:    19.123456789,
			Enabled:   true,
			SortOrder: 10,
		}},
	})
	require.NoError(t, err)
	require.True(t, config.Config.Enabled)
	require.NotNil(t, config.Config.EligibleSince)
	require.Len(t, config.Offers, 1)
	require.Equal(t, "新人首充", config.Offers[0].Name)
	require.InDelta(t, 10.00, config.Offers[0].Price, 0.00001)
	require.InDelta(t, 19.12345679, config.Offers[0].Amount, 0.000000001)

	_, err = svc.UpdateAdminConfig(context.Background(), UpdateFirstRechargeConfigInput{
		Enabled:          true,
		EligibilityScope: FirstRechargeEligibilityAllUsers,
		Offers:           []FirstRechargeOfferInput{},
	})
	require.Equal(t, "FIRST_RECHARGE_OFFER_INVALID", infraerrors.Reason(err))
}

type firstRechargeMemoryRepo struct {
	config    FirstRechargeConfig
	offers    []FirstRechargeOffer
	states    map[int64]*FirstRechargeUserState
	specified map[int64]bool
	nextID    int64
}

func newFirstRechargeMemoryRepo() *firstRechargeMemoryRepo {
	now := time.Now()
	return &firstRechargeMemoryRepo{
		config: FirstRechargeConfig{
			Enabled:          false,
			EligibilityScope: FirstRechargeEligibilityNewUsersAfterEnabled,
			CreatedAt:        now,
			UpdatedAt:        now,
		},
		states:    map[int64]*FirstRechargeUserState{},
		specified: map[int64]bool{},
		nextID:    1,
	}
}

func (r *firstRechargeMemoryRepo) GetConfig(context.Context) (*FirstRechargeConfig, error) {
	cfg := r.config
	return &cfg, nil
}

func (r *firstRechargeMemoryRepo) SaveConfig(_ context.Context, config FirstRechargeConfig, offers []FirstRechargeOfferInput) (*FirstRechargeAdminConfig, error) {
	now := time.Now()
	r.config = FirstRechargeConfig{
		Enabled:          config.Enabled,
		EligibilityScope: config.EligibilityScope,
		EligibleSince:    config.EligibleSince,
		CreatedAt:        r.config.CreatedAt,
		UpdatedAt:        now,
	}
	if r.config.CreatedAt.IsZero() {
		r.config.CreatedAt = now
	}

	nextOffers := make([]FirstRechargeOffer, 0, len(offers))
	for _, input := range offers {
		id := input.ID
		if id <= 0 {
			id = r.nextID
			r.nextID++
		}
		nextOffers = append(nextOffers, FirstRechargeOffer{
			ID:          id,
			Name:        input.Name,
			Description: input.Description,
			Price:       input.Price,
			Amount:      input.Amount,
			Enabled:     input.Enabled,
			SortOrder:   input.SortOrder,
			CreatedAt:   now,
			UpdatedAt:   now,
		})
	}
	r.offers = nextOffers
	return r.adminConfig(), nil
}

func (r *firstRechargeMemoryRepo) ListOffers(context.Context) ([]FirstRechargeOffer, error) {
	offers := append([]FirstRechargeOffer(nil), r.offers...)
	sort.SliceStable(offers, func(i, j int) bool {
		if offers[i].SortOrder == offers[j].SortOrder {
			return offers[i].ID < offers[j].ID
		}
		return offers[i].SortOrder < offers[j].SortOrder
	})
	return offers, nil
}

func (r *firstRechargeMemoryRepo) ListEnabledOffers(ctx context.Context) ([]FirstRechargeOffer, error) {
	offers, err := r.ListOffers(ctx)
	if err != nil {
		return nil, err
	}
	enabled := make([]FirstRechargeOffer, 0, len(offers))
	for _, offer := range offers {
		if offer.Enabled {
			enabled = append(enabled, offer)
		}
	}
	return enabled, nil
}

func (r *firstRechargeMemoryRepo) GetEnabledOfferByID(_ context.Context, offerID int64) (*FirstRechargeOffer, error) {
	for _, offer := range r.offers {
		if offer.ID == offerID && offer.Enabled {
			next := offer
			return &next, nil
		}
	}
	return nil, nil
}

func (r *firstRechargeMemoryRepo) GetUserState(_ context.Context, userID int64) (*FirstRechargeUserState, error) {
	state := r.states[userID]
	if state == nil {
		return nil, nil
	}
	cloned := *state
	return &cloned, nil
}

func (r *firstRechargeMemoryRepo) DismissPopup(_ context.Context, userID int64, dismissedAt time.Time) error {
	state := r.ensureState(userID)
	if state.PopupDismissedAt == nil {
		state.PopupDismissedAt = &dismissedAt
	}
	state.UpdatedAt = dismissedAt
	return nil
}

func (r *firstRechargeMemoryRepo) MarkCompleted(_ context.Context, userID, orderID int64, completedAt time.Time) error {
	state := r.ensureState(userID)
	if state.CompletedAt == nil {
		state.CompletedAt = &completedAt
		state.CompletedOrderID = &orderID
	}
	state.UpdatedAt = completedAt
	return nil
}

func (r *firstRechargeMemoryRepo) HasCompleted(_ context.Context, userID int64) (bool, error) {
	state := r.states[userID]
	return state != nil && state.CompletedAt != nil, nil
}

func (r *firstRechargeMemoryRepo) IsSpecifiedUser(_ context.Context, userID int64) (bool, error) {
	return r.specified[userID], nil
}

func (r *firstRechargeMemoryRepo) ListSpecifiedUsers(_ context.Context, params pagination.PaginationParams, search string) ([]FirstRechargeSpecifiedUser, *pagination.PaginationResult, error) {
	items := make([]FirstRechargeSpecifiedUser, 0, len(r.specified))
	for userID := range r.specified {
		item := FirstRechargeSpecifiedUser{UserID: userID, Email: "user@example.test", Username: "user", CreatedAt: time.Now()}
		if q := strings.TrimSpace(search); q != "" && !strings.Contains(item.Email, q) && !strings.Contains(item.Username, q) {
			continue
		}
		items = append(items, item)
	}
	return items, &pagination.PaginationResult{Total: int64(len(items)), Page: params.Page, PageSize: params.PageSize, Pages: 1}, nil
}

func (r *firstRechargeMemoryRepo) AddSpecifiedUser(_ context.Context, userID int64, _ *int64) error {
	r.specified[userID] = true
	return nil
}

func (r *firstRechargeMemoryRepo) RemoveSpecifiedUser(_ context.Context, userID int64) error {
	delete(r.specified, userID)
	return nil
}

func (r *firstRechargeMemoryRepo) adminConfig() *FirstRechargeAdminConfig {
	cfg := r.config
	offers := append([]FirstRechargeOffer(nil), r.offers...)
	return &FirstRechargeAdminConfig{Config: cfg, Offers: offers}
}

func (r *firstRechargeMemoryRepo) ensureState(userID int64) *FirstRechargeUserState {
	if state := r.states[userID]; state != nil {
		return state
	}
	now := time.Now()
	state := &FirstRechargeUserState{UserID: userID, CreatedAt: now, UpdatedAt: now}
	r.states[userID] = state
	return state
}

type firstRechargeUserRepoFake struct {
	users map[int64]*User
}

func (r *firstRechargeUserRepoFake) Create(context.Context, *User) error {
	panic("unexpected Create call")
}
func (r *firstRechargeUserRepoFake) GetByID(_ context.Context, id int64) (*User, error) {
	user := r.users[id]
	if user == nil {
		return nil, ErrUserNotFound
	}
	cloned := *user
	return &cloned, nil
}
func (r *firstRechargeUserRepoFake) GetByIDIncludeDeleted(ctx context.Context, id int64) (*User, error) {
	return r.GetByID(ctx, id)
}
func (r *firstRechargeUserRepoFake) GetByEmail(context.Context, string) (*User, error) {
	panic("unexpected GetByEmail call")
}
func (r *firstRechargeUserRepoFake) GetFirstAdmin(context.Context) (*User, error) {
	panic("unexpected GetFirstAdmin call")
}
func (r *firstRechargeUserRepoFake) Update(context.Context, *User) error {
	panic("unexpected Update call")
}
func (r *firstRechargeUserRepoFake) Delete(context.Context, int64) error {
	panic("unexpected Delete call")
}
func (r *firstRechargeUserRepoFake) GetUserAvatar(context.Context, int64) (*UserAvatar, error) {
	panic("unexpected GetUserAvatar call")
}
func (r *firstRechargeUserRepoFake) UpsertUserAvatar(context.Context, int64, UpsertUserAvatarInput) (*UserAvatar, error) {
	panic("unexpected UpsertUserAvatar call")
}
func (r *firstRechargeUserRepoFake) DeleteUserAvatar(context.Context, int64) error {
	panic("unexpected DeleteUserAvatar call")
}
func (r *firstRechargeUserRepoFake) List(context.Context, pagination.PaginationParams) ([]User, *pagination.PaginationResult, error) {
	panic("unexpected List call")
}
func (r *firstRechargeUserRepoFake) ListWithFilters(context.Context, pagination.PaginationParams, UserListFilters) ([]User, *pagination.PaginationResult, error) {
	panic("unexpected ListWithFilters call")
}
func (r *firstRechargeUserRepoFake) GetLatestUsedAtByUserIDs(context.Context, []int64) (map[int64]*time.Time, error) {
	panic("unexpected GetLatestUsedAtByUserIDs call")
}
func (r *firstRechargeUserRepoFake) GetLatestUsedAtByUserID(context.Context, int64) (*time.Time, error) {
	panic("unexpected GetLatestUsedAtByUserID call")
}
func (r *firstRechargeUserRepoFake) UpdateUserLastActiveAt(context.Context, int64, time.Time) error {
	panic("unexpected UpdateUserLastActiveAt call")
}
func (r *firstRechargeUserRepoFake) UpdateBalance(context.Context, int64, float64) error {
	panic("unexpected UpdateBalance call")
}
func (r *firstRechargeUserRepoFake) DeductBalance(context.Context, int64, float64) error {
	panic("unexpected DeductBalance call")
}
func (r *firstRechargeUserRepoFake) UpdateConcurrency(context.Context, int64, int) error {
	panic("unexpected UpdateConcurrency call")
}
func (r *firstRechargeUserRepoFake) BatchSetConcurrency(context.Context, []int64, int) (int, error) {
	panic("unexpected BatchSetConcurrency call")
}
func (r *firstRechargeUserRepoFake) BatchAddConcurrency(context.Context, []int64, int) (int, error) {
	panic("unexpected BatchAddConcurrency call")
}
func (r *firstRechargeUserRepoFake) ExistsByEmail(context.Context, string) (bool, error) {
	panic("unexpected ExistsByEmail call")
}
func (r *firstRechargeUserRepoFake) RemoveGroupFromAllowedGroups(context.Context, int64) (int64, error) {
	panic("unexpected RemoveGroupFromAllowedGroups call")
}
func (r *firstRechargeUserRepoFake) AddGroupToAllowedGroups(context.Context, int64, int64) error {
	panic("unexpected AddGroupToAllowedGroups call")
}
func (r *firstRechargeUserRepoFake) RemoveGroupFromUserAllowedGroups(context.Context, int64, int64) error {
	panic("unexpected RemoveGroupFromUserAllowedGroups call")
}
func (r *firstRechargeUserRepoFake) ListUserAuthIdentities(context.Context, int64) ([]UserAuthIdentityRecord, error) {
	panic("unexpected ListUserAuthIdentities call")
}
func (r *firstRechargeUserRepoFake) UnbindUserAuthProvider(context.Context, int64, string) error {
	panic("unexpected UnbindUserAuthProvider call")
}
func (r *firstRechargeUserRepoFake) UpdateTotpSecret(context.Context, int64, *string) error {
	panic("unexpected UpdateTotpSecret call")
}
func (r *firstRechargeUserRepoFake) EnableTotp(context.Context, int64) error {
	panic("unexpected EnableTotp call")
}
func (r *firstRechargeUserRepoFake) DisableTotp(context.Context, int64) error {
	panic("unexpected DisableTotp call")
}
