package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

const (
	defaultGroupRateScheduleStartTime     = "00:00"
	defaultGroupRateScheduleEndTime       = "00:00"
	defaultGroupRateSchedulePercent       = 100
	defaultGroupRateScheduleCheckInterval = time.Minute
	groupRateScheduleApplyTolerance       = 0.0000001
)

// GroupRateScheduleSettings describes the global daily multiplier override for all groups.
type GroupRateScheduleSettings struct {
	Enabled          bool               `json:"enabled"`
	StartTime        string             `json:"start_time"`
	EndTime          string             `json:"end_time"`
	Percent          int                `json:"percent"`
	Timezone         string             `json:"timezone"`
	Active           bool               `json:"active"`
	OriginalRates    map[string]float64 `json:"original_rates,omitempty"`
	LastAppliedAt    *time.Time         `json:"last_applied_at,omitempty"`
	LastRestoredAt   *time.Time         `json:"last_restored_at,omitempty"`
	LastTransitionAt *time.Time         `json:"last_transition_at,omitempty"`
}

// GroupRateScheduleService applies and restores group rate multiplier overrides.
type GroupRateScheduleService struct {
	settingRepo          SettingRepository
	groupRepo            GroupRepository
	authCacheInvalidator APIKeyAuthCacheInvalidator
	interval             time.Duration
	now                  func() time.Time
	stopCh               chan struct{}
	stopOnce             sync.Once
	wg                   sync.WaitGroup
	mu                   sync.Mutex
}

func NewGroupRateScheduleService(settingRepo SettingRepository, groupRepo GroupRepository, authCacheInvalidator APIKeyAuthCacheInvalidator) *GroupRateScheduleService {
	return &GroupRateScheduleService{
		settingRepo:          settingRepo,
		groupRepo:            groupRepo,
		authCacheInvalidator: authCacheInvalidator,
		interval:             defaultGroupRateScheduleCheckInterval,
		now:                  time.Now,
		stopCh:               make(chan struct{}),
	}
}

func ProvideGroupRateScheduleService(settingRepo SettingRepository, groupRepo GroupRepository, authCacheInvalidator APIKeyAuthCacheInvalidator) *GroupRateScheduleService {
	svc := NewGroupRateScheduleService(settingRepo, groupRepo, authCacheInvalidator)
	svc.Start()
	return svc
}

func (s *GroupRateScheduleService) Start() {
	if s == nil || s.settingRepo == nil || s.groupRepo == nil || s.interval <= 0 {
		return
	}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		s.runOnce()
		for {
			select {
			case <-ticker.C:
				s.runOnce()
			case <-s.stopCh:
				return
			}
		}
	}()
}

func (s *GroupRateScheduleService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		close(s.stopCh)
	})
	s.wg.Wait()
}

func (s *GroupRateScheduleService) GetSettings(ctx context.Context) (*GroupRateScheduleSettings, error) {
	if s == nil || s.settingRepo == nil {
		return DefaultGroupRateScheduleSettings(), nil
	}
	return loadGroupRateScheduleSettings(ctx, s.settingRepo)
}

func (s *GroupRateScheduleService) UpdateSettings(ctx context.Context, input *GroupRateScheduleSettings) (*GroupRateScheduleSettings, error) {
	if s == nil || s.settingRepo == nil {
		return nil, errors.New("group rate schedule service is unavailable")
	}
	normalized, err := normalizeGroupRateScheduleSettings(input)
	if err != nil {
		return nil, err
	}

	current, err := loadGroupRateScheduleSettings(ctx, s.settingRepo)
	if err != nil {
		return nil, err
	}
	normalized.Active = current.Active
	normalized.OriginalRates = copyRateMap(current.OriginalRates)
	normalized.LastAppliedAt = cloneGroupRateScheduleTimePtr(current.LastAppliedAt)
	normalized.LastRestoredAt = cloneGroupRateScheduleTimePtr(current.LastRestoredAt)
	normalized.LastTransitionAt = cloneGroupRateScheduleTimePtr(current.LastTransitionAt)

	if err := saveGroupRateScheduleSettings(ctx, s.settingRepo, normalized); err != nil {
		return nil, err
	}
	if err := s.evaluate(ctx); err != nil {
		return nil, err
	}
	updated, err := s.GetSettings(ctx)
	if err != nil {
		return nil, err
	}
	desiredActive, err := updated.inWindow(s.now())
	if err != nil {
		return nil, err
	}
	if updated.Enabled && desiredActive && !updated.Active {
		return nil, errors.New("settings saved but group rate schedule was not applied")
	}
	if (!updated.Enabled || !desiredActive) && updated.Active {
		return nil, errors.New("settings saved but group rate schedule was not restored")
	}
	return updated, nil
}

func (s *GroupRateScheduleService) runOnce() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := s.evaluate(ctx); err != nil {
		log.Printf("[GroupRateSchedule] evaluate failed: %v", err)
	}
}

func (s *GroupRateScheduleService) evaluate(ctx context.Context) error {
	if s == nil || s.settingRepo == nil || s.groupRepo == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	settings, err := loadGroupRateScheduleSettings(ctx, s.settingRepo)
	if err != nil {
		return err
	}
	activeNow, err := settings.inWindow(s.now())
	if err != nil {
		return err
	}
	if settings.Enabled && activeNow {
		return s.apply(ctx, settings)
	}
	if settings.Active {
		return s.restore(ctx, settings)
	}
	return nil
}

func (s *GroupRateScheduleService) apply(ctx context.Context, settings *GroupRateScheduleSettings) error {
	groups, err := s.listAllGroups(ctx)
	if err != nil {
		return fmt.Errorf("list groups: %w", err)
	}

	discount := float64(settings.Percent) / 100
	originalRates := copyRateMap(settings.OriginalRates)
	if originalRates == nil {
		originalRates = make(map[string]float64, len(groups))
	}
	updates := make([]GroupRateMultiplierUpdate, 0, len(groups))
	updatedGroupIDs := make([]int64, 0, len(groups))
	changedOriginalRates := false
	for i := range groups {
		group := groups[i]
		key := strconv.FormatInt(group.ID, 10)
		if _, exists := originalRates[key]; !exists {
			originalRates[key] = group.RateMultiplier
			changedOriginalRates = true
		}
		target := originalRates[key] * discount
		if math.Abs(group.RateMultiplier-target) > groupRateScheduleApplyTolerance {
			updates = append(updates, GroupRateMultiplierUpdate{
				ID:             group.ID,
				RateMultiplier: target,
			})
			updatedGroupIDs = append(updatedGroupIDs, group.ID)
		}
	}
	if settings.Active && len(updates) == 0 && !changedOriginalRates {
		return nil
	}
	if len(updates) > 0 {
		if err := s.groupRepo.UpdateRateMultipliers(ctx, updates); err != nil {
			return fmt.Errorf("apply group rate schedule: %w", err)
		}
	}

	now := s.now()
	next := cloneGroupRateScheduleSettings(settings)
	next.Active = true
	next.OriginalRates = originalRates
	if len(updates) > 0 || changedOriginalRates || !settings.Active {
		next.LastAppliedAt = &now
	}
	if !settings.Active {
		next.LastTransitionAt = &now
	}
	if err := saveGroupRateScheduleSettings(ctx, s.settingRepo, next); err != nil {
		return err
	}
	s.invalidateGroups(ctx, updatedGroupIDs)
	return nil
}

func (s *GroupRateScheduleService) listAllGroups(ctx context.Context) ([]Group, error) {
	allGroups := make([]Group, 0)
	for page := 1; ; page++ {
		groups, result, err := s.groupRepo.ListWithFilters(
			ctx,
			pagination.PaginationParams{Page: page, PageSize: 1000, SortBy: "id", SortOrder: pagination.SortOrderAsc},
			"",
			"",
			"",
			nil,
		)
		if err != nil {
			return nil, err
		}
		allGroups = append(allGroups, groups...)
		if result == nil || page >= result.Pages || len(groups) == 0 {
			return allGroups, nil
		}
	}
}

func (s *GroupRateScheduleService) restore(ctx context.Context, settings *GroupRateScheduleSettings) error {
	groups, err := s.listAllGroups(ctx)
	if err != nil {
		return fmt.Errorf("list groups: %w", err)
	}
	existingGroupIDs := make(map[int64]struct{}, len(groups))
	for i := range groups {
		existingGroupIDs[groups[i].ID] = struct{}{}
	}

	updates := make([]GroupRateMultiplierUpdate, 0, len(settings.OriginalRates))
	groupIDs := make([]int64, 0, len(settings.OriginalRates))
	keys := make([]string, 0, len(settings.OriginalRates))
	for key := range settings.OriginalRates {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		groupID, err := strconv.ParseInt(key, 10, 64)
		if err != nil || groupID <= 0 {
			continue
		}
		if _, exists := existingGroupIDs[groupID]; !exists {
			continue
		}
		rate := settings.OriginalRates[key]
		if rate <= 0 {
			continue
		}
		updates = append(updates, GroupRateMultiplierUpdate{ID: groupID, RateMultiplier: rate})
		groupIDs = append(groupIDs, groupID)
	}
	if len(updates) > 0 {
		if err := s.groupRepo.UpdateRateMultipliers(ctx, updates); err != nil {
			return fmt.Errorf("restore group rate schedule: %w", err)
		}
	}

	now := s.now()
	next := cloneGroupRateScheduleSettings(settings)
	next.Active = false
	next.OriginalRates = nil
	next.LastRestoredAt = &now
	next.LastTransitionAt = &now
	if err := saveGroupRateScheduleSettings(ctx, s.settingRepo, next); err != nil {
		return err
	}
	s.invalidateGroups(ctx, groupIDs)
	return nil
}

func (s *GroupRateScheduleService) invalidateGroups(ctx context.Context, groupIDs []int64) {
	if s.authCacheInvalidator == nil {
		return
	}
	seen := make(map[int64]struct{}, len(groupIDs))
	for _, groupID := range groupIDs {
		if groupID <= 0 {
			continue
		}
		if _, ok := seen[groupID]; ok {
			continue
		}
		seen[groupID] = struct{}{}
		s.authCacheInvalidator.InvalidateAuthCacheByGroupID(ctx, groupID)
	}
}

func DefaultGroupRateScheduleSettings() *GroupRateScheduleSettings {
	return &GroupRateScheduleSettings{
		Enabled:       false,
		StartTime:     defaultGroupRateScheduleStartTime,
		EndTime:       defaultGroupRateScheduleEndTime,
		Percent:       defaultGroupRateSchedulePercent,
		Timezone:      "",
		Active:        false,
		OriginalRates: nil,
	}
}

func loadGroupRateScheduleSettings(ctx context.Context, repo SettingRepository) (*GroupRateScheduleSettings, error) {
	value, err := repo.GetValue(ctx, SettingKeyGroupRateScheduleSettings)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			return DefaultGroupRateScheduleSettings(), nil
		}
		return nil, fmt.Errorf("get group rate schedule settings: %w", err)
	}
	var settings GroupRateScheduleSettings
	if err := json.Unmarshal([]byte(value), &settings); err != nil {
		return nil, fmt.Errorf("unmarshal group rate schedule settings: %w", err)
	}
	normalized, err := normalizeGroupRateScheduleSettings(&settings)
	if err != nil {
		return nil, err
	}
	normalized.Active = settings.Active
	normalized.OriginalRates = copyRateMap(settings.OriginalRates)
	normalized.LastAppliedAt = cloneGroupRateScheduleTimePtr(settings.LastAppliedAt)
	normalized.LastRestoredAt = cloneGroupRateScheduleTimePtr(settings.LastRestoredAt)
	normalized.LastTransitionAt = cloneGroupRateScheduleTimePtr(settings.LastTransitionAt)
	return normalized, nil
}

func saveGroupRateScheduleSettings(ctx context.Context, repo SettingRepository, settings *GroupRateScheduleSettings) error {
	normalized, err := normalizeGroupRateScheduleSettings(settings)
	if err != nil {
		return err
	}
	normalized.Active = settings.Active
	normalized.OriginalRates = copyRateMap(settings.OriginalRates)
	normalized.LastAppliedAt = cloneGroupRateScheduleTimePtr(settings.LastAppliedAt)
	normalized.LastRestoredAt = cloneGroupRateScheduleTimePtr(settings.LastRestoredAt)
	normalized.LastTransitionAt = cloneGroupRateScheduleTimePtr(settings.LastTransitionAt)

	data, err := json.Marshal(normalized)
	if err != nil {
		return fmt.Errorf("marshal group rate schedule settings: %w", err)
	}
	return repo.Set(ctx, SettingKeyGroupRateScheduleSettings, string(data))
}

func normalizeGroupRateScheduleSettings(input *GroupRateScheduleSettings) (*GroupRateScheduleSettings, error) {
	if input == nil {
		return nil, infraerrors.BadRequest("INVALID_GROUP_RATE_SCHEDULE", "settings cannot be nil")
	}
	startTime := strings.TrimSpace(input.StartTime)
	if startTime == "" {
		startTime = defaultGroupRateScheduleStartTime
	}
	endTime := strings.TrimSpace(input.EndTime)
	if endTime == "" {
		endTime = defaultGroupRateScheduleEndTime
	}
	if _, err := parseHHMM(startTime); err != nil {
		return nil, err
	}
	if _, err := parseHHMM(endTime); err != nil {
		return nil, err
	}
	if input.Enabled && startTime == endTime {
		return nil, infraerrors.BadRequest("INVALID_GROUP_RATE_SCHEDULE", "start_time and end_time cannot be the same when enabled")
	}
	percent := input.Percent
	if percent == 0 {
		percent = defaultGroupRateSchedulePercent
	}
	if percent < 1 || percent > 100 {
		return nil, infraerrors.BadRequest("INVALID_GROUP_RATE_SCHEDULE", "percent must be between 1 and 100")
	}
	return &GroupRateScheduleSettings{
		Enabled:          input.Enabled,
		StartTime:        startTime,
		EndTime:          endTime,
		Percent:          percent,
		Timezone:         strings.TrimSpace(input.Timezone),
		Active:           input.Active,
		OriginalRates:    copyRateMap(input.OriginalRates),
		LastAppliedAt:    cloneGroupRateScheduleTimePtr(input.LastAppliedAt),
		LastRestoredAt:   cloneGroupRateScheduleTimePtr(input.LastRestoredAt),
		LastTransitionAt: cloneGroupRateScheduleTimePtr(input.LastTransitionAt),
	}, nil
}

func (s *GroupRateScheduleSettings) inWindow(now time.Time) (bool, error) {
	if s == nil || !s.Enabled {
		return false, nil
	}
	loc := now.Location()
	if s.Timezone != "" {
		loaded, err := time.LoadLocation(s.Timezone)
		if err != nil {
			return false, fmt.Errorf("invalid timezone: %w", err)
		}
		loc = loaded
	}
	localNow := now.In(loc)
	currentMinute := localNow.Hour()*60 + localNow.Minute()
	startMinute, err := parseHHMM(s.StartTime)
	if err != nil {
		return false, err
	}
	endMinute, err := parseHHMM(s.EndTime)
	if err != nil {
		return false, err
	}
	if startMinute == endMinute {
		return false, nil
	}
	if startMinute < endMinute {
		return currentMinute >= startMinute && currentMinute < endMinute, nil
	}
	return currentMinute >= startMinute || currentMinute < endMinute, nil
}

func parseHHMM(value string) (int, error) {
	if len(value) != 5 || value[2] != ':' {
		return 0, infraerrors.BadRequest("INVALID_GROUP_RATE_SCHEDULE", "time must use HH:mm format")
	}
	hour, err := strconv.Atoi(value[:2])
	if err != nil || hour < 0 || hour > 23 {
		return 0, infraerrors.BadRequest("INVALID_GROUP_RATE_SCHEDULE", "time hour must be between 00 and 23")
	}
	minute, err := strconv.Atoi(value[3:])
	if err != nil || minute < 0 || minute > 59 {
		return 0, infraerrors.BadRequest("INVALID_GROUP_RATE_SCHEDULE", "time minute must be between 00 and 59")
	}
	return hour*60 + minute, nil
}

func cloneGroupRateScheduleSettings(input *GroupRateScheduleSettings) *GroupRateScheduleSettings {
	if input == nil {
		return DefaultGroupRateScheduleSettings()
	}
	return &GroupRateScheduleSettings{
		Enabled:          input.Enabled,
		StartTime:        input.StartTime,
		EndTime:          input.EndTime,
		Percent:          input.Percent,
		Timezone:         input.Timezone,
		Active:           input.Active,
		OriginalRates:    copyRateMap(input.OriginalRates),
		LastAppliedAt:    cloneGroupRateScheduleTimePtr(input.LastAppliedAt),
		LastRestoredAt:   cloneGroupRateScheduleTimePtr(input.LastRestoredAt),
		LastTransitionAt: cloneGroupRateScheduleTimePtr(input.LastTransitionAt),
	}
}

func copyRateMap(input map[string]float64) map[string]float64 {
	if len(input) == 0 {
		return nil
	}
	out := make(map[string]float64, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func cloneGroupRateScheduleTimePtr(input *time.Time) *time.Time {
	if input == nil {
		return nil
	}
	out := *input
	return &out
}
