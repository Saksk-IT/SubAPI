package service

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

const (
	FirstRechargeActivityType = "first_recharge"

	FirstRechargeAnnouncementID int64 = -1001

	FirstRechargeEligibilityNewUsersAfterEnabled = "new_users_after_enabled"
	FirstRechargeEligibilityAllUsers             = "all_users"
	FirstRechargeEligibilitySpecifiedUsers       = "specified_users"

	maxFirstRechargeOffers = 20
)

var (
	ErrFirstRechargeConfigInvalid = infraerrors.BadRequest("FIRST_RECHARGE_CONFIG_INVALID", "first recharge config is invalid")
	ErrFirstRechargeOfferInvalid  = infraerrors.BadRequest("FIRST_RECHARGE_OFFER_INVALID", "first recharge offer is invalid")
	ErrFirstRechargeOfferNotFound = infraerrors.NotFound("FIRST_RECHARGE_OFFER_NOT_FOUND", "first recharge offer not found")
	ErrFirstRechargeUnavailable   = infraerrors.Forbidden("FIRST_RECHARGE_UNAVAILABLE", "first recharge is unavailable")
	ErrFirstRechargeCompleted     = infraerrors.Conflict("FIRST_RECHARGE_COMPLETED", "first recharge has already been completed")
)

type FirstRechargeConfig struct {
	Enabled          bool       `json:"enabled"`
	EligibilityScope string     `json:"eligibility_scope"`
	EligibleSince    *time.Time `json:"eligible_since,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type FirstRechargeOffer struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Price       float64   `json:"price"`
	Amount      float64   `json:"amount"`
	Enabled     bool      `json:"enabled"`
	SortOrder   int       `json:"sort_order"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type FirstRechargeUserState struct {
	UserID           int64      `json:"user_id"`
	PopupDismissedAt *time.Time `json:"popup_dismissed_at,omitempty"`
	CompletedOrderID *int64     `json:"completed_order_id,omitempty"`
	CompletedAt      *time.Time `json:"completed_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type FirstRechargeStatus struct {
	Enabled          bool                 `json:"enabled"`
	Eligible         bool                 `json:"eligible"`
	Completed        bool                 `json:"completed"`
	PopupDismissed   bool                 `json:"popup_dismissed"`
	EligibilityScope string               `json:"eligibility_scope"`
	EligibleSince    *time.Time           `json:"eligible_since,omitempty"`
	CompletedAt      *time.Time           `json:"completed_at,omitempty"`
	Offers           []FirstRechargeOffer `json:"offers"`
}

type FirstRechargeAdminConfig struct {
	Config FirstRechargeConfig  `json:"config"`
	Offers []FirstRechargeOffer `json:"offers"`
}

type FirstRechargeOfferInput struct {
	ID          int64   `json:"id,omitempty"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Amount      float64 `json:"amount"`
	Enabled     bool    `json:"enabled"`
	SortOrder   int     `json:"sort_order"`
}

type UpdateFirstRechargeConfigInput struct {
	Enabled          bool
	EligibilityScope string
	Offers           []FirstRechargeOfferInput
}

type FirstRechargeSpecifiedUser struct {
	UserID    int64     `json:"user_id"`
	Email     string    `json:"email"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
}

type FirstRechargeRepository interface {
	GetConfig(ctx context.Context) (*FirstRechargeConfig, error)
	SaveConfig(ctx context.Context, config FirstRechargeConfig, offers []FirstRechargeOfferInput) (*FirstRechargeAdminConfig, error)
	ListOffers(ctx context.Context) ([]FirstRechargeOffer, error)
	ListEnabledOffers(ctx context.Context) ([]FirstRechargeOffer, error)
	GetEnabledOfferByID(ctx context.Context, offerID int64) (*FirstRechargeOffer, error)
	GetUserState(ctx context.Context, userID int64) (*FirstRechargeUserState, error)
	DismissPopup(ctx context.Context, userID int64, dismissedAt time.Time) error
	MarkCompleted(ctx context.Context, userID, orderID int64, completedAt time.Time) error
	HasCompleted(ctx context.Context, userID int64) (bool, error)
	IsSpecifiedUser(ctx context.Context, userID int64) (bool, error)
	ListSpecifiedUsers(ctx context.Context, params pagination.PaginationParams, search string) ([]FirstRechargeSpecifiedUser, *pagination.PaginationResult, error)
	AddSpecifiedUser(ctx context.Context, userID int64, actorID *int64) error
	RemoveSpecifiedUser(ctx context.Context, userID int64) error
}

type FirstRechargeActivityService struct {
	repo     FirstRechargeRepository
	userRepo UserRepository
}

func NewFirstRechargeActivityService(repo FirstRechargeRepository, userRepo UserRepository) *FirstRechargeActivityService {
	return &FirstRechargeActivityService{repo: repo, userRepo: userRepo}
}

func (s *FirstRechargeActivityService) GetAdminConfig(ctx context.Context) (*FirstRechargeAdminConfig, error) {
	config, err := s.repo.GetConfig(ctx)
	if err != nil {
		return nil, err
	}
	offers, err := s.repo.ListOffers(ctx)
	if err != nil {
		return nil, err
	}
	return &FirstRechargeAdminConfig{Config: *config, Offers: offers}, nil
}

func (s *FirstRechargeActivityService) UpdateAdminConfig(ctx context.Context, input UpdateFirstRechargeConfigInput) (*FirstRechargeAdminConfig, error) {
	scope := normalizeFirstRechargeScope(input.EligibilityScope)
	if !isValidFirstRechargeScope(scope) {
		return nil, ErrFirstRechargeConfigInvalid
	}
	if len(input.Offers) > maxFirstRechargeOffers {
		return nil, ErrFirstRechargeOfferInvalid
	}

	now := time.Now()
	current, err := s.repo.GetConfig(ctx)
	if err != nil {
		return nil, err
	}
	eligibleSince := current.EligibleSince
	if input.Enabled && scope == FirstRechargeEligibilityNewUsersAfterEnabled {
		if !current.Enabled || current.EligibilityScope != scope || eligibleSince == nil {
			eligibleSince = &now
		}
	}
	if scope != FirstRechargeEligibilityNewUsersAfterEnabled {
		eligibleSince = nil
	}

	offers := make([]FirstRechargeOfferInput, 0, len(input.Offers))
	enabledOfferCount := 0
	for i := range input.Offers {
		offer, err := normalizeFirstRechargeOfferInput(input.Offers[i])
		if err != nil {
			return nil, err
		}
		if offer.Enabled {
			enabledOfferCount++
		}
		offers = append(offers, offer)
	}
	if input.Enabled && enabledOfferCount == 0 {
		return nil, ErrFirstRechargeOfferInvalid
	}
	sort.SliceStable(offers, func(i, j int) bool {
		if offers[i].SortOrder == offers[j].SortOrder {
			return offers[i].ID < offers[j].ID
		}
		return offers[i].SortOrder < offers[j].SortOrder
	})

	return s.repo.SaveConfig(ctx, FirstRechargeConfig{
		Enabled:          input.Enabled,
		EligibilityScope: scope,
		EligibleSince:    eligibleSince,
		UpdatedAt:        now,
	}, offers)
}

func (s *FirstRechargeActivityService) GetStatus(ctx context.Context, userID int64) (*FirstRechargeStatus, error) {
	config, err := s.repo.GetConfig(ctx)
	if err != nil {
		return nil, err
	}
	state, err := s.repo.GetUserState(ctx, userID)
	if err != nil {
		return nil, err
	}
	completed := state != nil && state.CompletedAt != nil
	status := &FirstRechargeStatus{
		Enabled:          config.Enabled,
		Eligible:         false,
		Completed:        completed,
		PopupDismissed:   state != nil && state.PopupDismissedAt != nil,
		EligibilityScope: config.EligibilityScope,
		EligibleSince:    config.EligibleSince,
	}
	if completed {
		status.CompletedAt = state.CompletedAt
	}
	if !config.Enabled || completed {
		return status, nil
	}
	eligible, err := s.isUserEligible(ctx, userID, config)
	if err != nil {
		return nil, err
	}
	status.Eligible = eligible
	if !eligible {
		return status, nil
	}
	offers, err := s.repo.ListEnabledOffers(ctx)
	if err != nil {
		return nil, err
	}
	status.Offers = offers
	if len(status.Offers) == 0 {
		status.Eligible = false
	}
	return status, nil
}

func (s *FirstRechargeActivityService) DismissPopup(ctx context.Context, userID int64) error {
	status, err := s.GetStatus(ctx, userID)
	if err != nil {
		return err
	}
	if !status.Enabled || !status.Eligible || status.Completed {
		return nil
	}
	return s.repo.DismissPopup(ctx, userID, time.Now())
}

func (s *FirstRechargeActivityService) PrepareOrder(ctx context.Context, userID, offerID int64) (*FirstRechargeOffer, error) {
	if offerID <= 0 {
		return nil, ErrFirstRechargeOfferNotFound
	}
	status, err := s.GetStatus(ctx, userID)
	if err != nil {
		return nil, err
	}
	if status.Completed {
		return nil, ErrFirstRechargeCompleted
	}
	if !status.Enabled || !status.Eligible {
		return nil, ErrFirstRechargeUnavailable
	}
	offer, err := s.repo.GetEnabledOfferByID(ctx, offerID)
	if err != nil {
		return nil, err
	}
	if offer == nil {
		return nil, ErrFirstRechargeOfferNotFound
	}
	return offer, nil
}

func (s *FirstRechargeActivityService) MarkCompleted(ctx context.Context, userID, orderID int64) error {
	if userID <= 0 || orderID <= 0 {
		return nil
	}
	return s.repo.MarkCompleted(ctx, userID, orderID, time.Now())
}

func (s *FirstRechargeActivityService) ListSpecifiedUsers(ctx context.Context, params pagination.PaginationParams, search string) ([]FirstRechargeSpecifiedUser, *pagination.PaginationResult, error) {
	return s.repo.ListSpecifiedUsers(ctx, params, strings.TrimSpace(search))
}

func (s *FirstRechargeActivityService) AddSpecifiedUser(ctx context.Context, userID int64, actorID *int64) error {
	if userID <= 0 {
		return infraerrors.BadRequest("INVALID_USER", "invalid user")
	}
	if _, err := s.userRepo.GetByID(ctx, userID); err != nil {
		return err
	}
	return s.repo.AddSpecifiedUser(ctx, userID, actorID)
}

func (s *FirstRechargeActivityService) RemoveSpecifiedUser(ctx context.Context, userID int64) error {
	if userID <= 0 {
		return infraerrors.BadRequest("INVALID_USER", "invalid user")
	}
	return s.repo.RemoveSpecifiedUser(ctx, userID)
}

func (s *FirstRechargeActivityService) isUserEligible(ctx context.Context, userID int64, config *FirstRechargeConfig) (bool, error) {
	if config == nil || !config.Enabled {
		return false, nil
	}
	switch config.EligibilityScope {
	case FirstRechargeEligibilityAllUsers:
		return true, nil
	case FirstRechargeEligibilitySpecifiedUsers:
		return s.repo.IsSpecifiedUser(ctx, userID)
	case FirstRechargeEligibilityNewUsersAfterEnabled:
		if config.EligibleSince == nil {
			return false, nil
		}
		user, err := s.userRepo.GetByID(ctx, userID)
		if err != nil {
			return false, err
		}
		return !user.CreatedAt.Before(*config.EligibleSince), nil
	default:
		return false, nil
	}
}

func normalizeFirstRechargeScope(scope string) string {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return FirstRechargeEligibilityNewUsersAfterEnabled
	}
	return scope
}

func isValidFirstRechargeScope(scope string) bool {
	switch scope {
	case FirstRechargeEligibilityNewUsersAfterEnabled, FirstRechargeEligibilityAllUsers, FirstRechargeEligibilitySpecifiedUsers:
		return true
	default:
		return false
	}
}

func normalizeFirstRechargeOfferInput(input FirstRechargeOfferInput) (FirstRechargeOfferInput, error) {
	name := strings.TrimSpace(input.Name)
	description := strings.TrimSpace(input.Description)
	if len(name) > 100 || len(description) > 1000 {
		return FirstRechargeOfferInput{}, ErrFirstRechargeOfferInvalid
	}
	if input.Price <= 0 || input.Amount <= 0 || math.IsNaN(input.Price) || math.IsNaN(input.Amount) || math.IsInf(input.Price, 0) || math.IsInf(input.Amount, 0) {
		return FirstRechargeOfferInput{}, ErrFirstRechargeOfferInvalid
	}
	if input.ID < 0 {
		return FirstRechargeOfferInput{}, ErrFirstRechargeOfferInvalid
	}
	if name == "" {
		name = fmt.Sprintf("%.2f", input.Price)
	}
	return FirstRechargeOfferInput{
		ID:          input.ID,
		Name:        name,
		Description: description,
		Price:       roundTo(input.Price, 2),
		Amount:      roundTo(input.Amount, 8),
		Enabled:     input.Enabled,
		SortOrder:   input.SortOrder,
	}, nil
}
