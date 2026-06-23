package service

import "testing"

func TestCompactChannelMonitorSortUpdatesKeepsLastOrder(t *testing.T) {
	got := compactChannelMonitorSortUpdates([]ChannelMonitorSortOrderUpdate{
		{ID: 1, SortOrder: 10},
		{ID: 0, SortOrder: 20},
		{ID: 2, SortOrder: 30},
		{ID: 1, SortOrder: 40},
		{ID: -3, SortOrder: 50},
	})

	if len(got) != 2 {
		t.Fatalf("expected 2 updates, got %d: %#v", len(got), got)
	}
	if got[0] != (ChannelMonitorSortOrderUpdate{ID: 1, SortOrder: 40}) {
		t.Fatalf("first update = %#v, want id=1 sort_order=40", got[0])
	}
	if got[1] != (ChannelMonitorSortOrderUpdate{ID: 2, SortOrder: 30}) {
		t.Fatalf("second update = %#v, want id=2 sort_order=30", got[1])
	}
}
