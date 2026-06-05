package service

import (
	"context"
	"testing"
	"time"
)

type visibilityMonitorRepo struct {
	monitors []*ChannelMonitor
}

func (r *visibilityMonitorRepo) Create(context.Context, *ChannelMonitor) error       { return nil }
func (r *visibilityMonitorRepo) Update(context.Context, *ChannelMonitor) error       { return nil }
func (r *visibilityMonitorRepo) Delete(context.Context, int64) error                 { return nil }
func (r *visibilityMonitorRepo) MarkChecked(context.Context, int64, time.Time) error { return nil }
func (r *visibilityMonitorRepo) InsertHistoryBatch(context.Context, []*ChannelMonitorHistoryRow) error {
	return nil
}
func (r *visibilityMonitorRepo) DeleteHistoryBefore(context.Context, time.Time) (int64, error) {
	return 0, nil
}
func (r *visibilityMonitorRepo) UpsertDailyRollupsFor(context.Context, time.Time) (int64, error) {
	return 0, nil
}
func (r *visibilityMonitorRepo) DeleteRollupsBefore(context.Context, time.Time) (int64, error) {
	return 0, nil
}
func (r *visibilityMonitorRepo) LoadAggregationWatermark(context.Context) (*time.Time, error) {
	return nil, nil
}
func (r *visibilityMonitorRepo) UpdateAggregationWatermark(context.Context, time.Time) error {
	return nil
}

func (r *visibilityMonitorRepo) GetByID(_ context.Context, id int64) (*ChannelMonitor, error) {
	for _, m := range r.monitors {
		if m.ID == id {
			return m, nil
		}
	}
	return nil, ErrChannelMonitorNotFound
}

func (r *visibilityMonitorRepo) List(_ context.Context, _ ChannelMonitorListParams) ([]*ChannelMonitor, int64, error) {
	return r.monitors, int64(len(r.monitors)), nil
}

func (r *visibilityMonitorRepo) ListEnabled(context.Context) ([]*ChannelMonitor, error) {
	out := make([]*ChannelMonitor, 0, len(r.monitors))
	for _, m := range r.monitors {
		if m.Enabled {
			out = append(out, m)
		}
	}
	return out, nil
}

func (r *visibilityMonitorRepo) ListHistory(context.Context, int64, string, int) ([]*ChannelMonitorHistoryEntry, error) {
	return nil, nil
}

func (r *visibilityMonitorRepo) ListLatestPerModel(context.Context, int64) ([]*ChannelMonitorLatest, error) {
	return nil, nil
}

func (r *visibilityMonitorRepo) ComputeAvailability(context.Context, int64, int) ([]*ChannelMonitorAvailability, error) {
	return nil, nil
}

func (r *visibilityMonitorRepo) ListLatestForMonitorIDs(context.Context, []int64) (map[int64][]*ChannelMonitorLatest, error) {
	return map[int64][]*ChannelMonitorLatest{}, nil
}

func (r *visibilityMonitorRepo) ComputeAvailabilityForMonitors(context.Context, []int64, int) (map[int64][]*ChannelMonitorAvailability, error) {
	return map[int64][]*ChannelMonitorAvailability{}, nil
}

func (r *visibilityMonitorRepo) ListRecentHistoryForMonitors(context.Context, []int64, map[int64]string, int) (map[int64][]*ChannelMonitorHistoryEntry, error) {
	return map[int64][]*ChannelMonitorHistoryEntry{}, nil
}

func newVisibilityMonitor(id int64, name string, userVisible bool) *ChannelMonitor {
	return &ChannelMonitor{
		ID:              id,
		Name:            name,
		Provider:        MonitorProviderOpenAI,
		APIMode:         MonitorAPIModeChatCompletions,
		Endpoint:        "https://api.example.com",
		APIKey:          "encrypted",
		PrimaryModel:    "gpt-4o-mini",
		Enabled:         true,
		UserVisible:     userVisible,
		IntervalSeconds: 60,
	}
}

func TestListUserViewSkipsInvisibleMonitors(t *testing.T) {
	repo := &visibilityMonitorRepo{monitors: []*ChannelMonitor{
		newVisibilityMonitor(1, "visible", true),
		newVisibilityMonitor(2, "hidden", false),
	}}
	svc := NewChannelMonitorService(repo, nil)

	views, err := svc.ListUserView(context.Background())
	if err != nil {
		t.Fatalf("ListUserView returned error: %v", err)
	}
	if len(views) != 1 {
		t.Fatalf("expected 1 visible monitor, got %d", len(views))
	}
	if views[0].ID != 1 {
		t.Fatalf("expected visible monitor id=1, got %d", views[0].ID)
	}
}

func TestGetUserDetailRejectsInvisibleMonitor(t *testing.T) {
	repo := &visibilityMonitorRepo{monitors: []*ChannelMonitor{
		newVisibilityMonitor(2, "hidden", false),
	}}
	svc := NewChannelMonitorService(repo, nil)

	if _, err := svc.GetUserDetail(context.Background(), 2); err != ErrChannelMonitorNotFound {
		t.Fatalf("expected ErrChannelMonitorNotFound, got %v", err)
	}
}
