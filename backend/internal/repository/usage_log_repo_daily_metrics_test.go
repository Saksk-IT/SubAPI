package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestUsageLogRepositoryGetDailyMetrics(t *testing.T) {
	t.Setenv("TZ", "UTC")
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}

	start := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 4, 0, 0, 0, 0, time.UTC)

	mock.ExpectQuery(`(?s)FROM\s+usage_dashboard_daily.*ORDER BY bucket_date ASC`).
		WithArgs(start, end).
		WillReturnRows(sqlmock.NewRows([]string{"date", "total_tokens", "active_users"}).
			AddRow("2026-03-01", int64(130), int64(4)).
			AddRow("2026-03-03", int64(570), int64(7)))

	mock.ExpectQuery(`(?s)FROM\s+users.*deleted_at IS NULL.*ORDER BY date ASC`).
		WithArgs(start, end, "UTC").
		WillReturnRows(sqlmock.NewRows([]string{"date", "new_users"}).
			AddRow("2026-03-01", int64(2)).
			AddRow("2026-03-03", int64(1)))

	got, err := repo.GetDailyMetrics(context.Background(), start, end)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())

	require.Equal(t, "2026-03-01", got.StartDate)
	require.Equal(t, "2026-03-03", got.EndDate)
	require.Len(t, got.Series, 3)
	require.Equal(t, "2026-03-01", got.Series[0].Date)
	require.Equal(t, int64(130), got.Series[0].TotalTokens)
	require.Equal(t, int64(2), got.Series[0].NewUsers)
	require.Equal(t, int64(4), got.Series[0].ActiveUsers)
	require.Equal(t, "2026-03-02", got.Series[1].Date)
	require.Equal(t, int64(0), got.Series[1].TotalTokens)
	require.Equal(t, int64(0), got.Series[1].NewUsers)
	require.Equal(t, int64(0), got.Series[1].ActiveUsers)
	require.Equal(t, "2026-03-03", got.Series[2].Date)
	require.Equal(t, int64(570), got.Series[2].TotalTokens)
	require.Equal(t, int64(1), got.Series[2].NewUsers)
	require.Equal(t, int64(7), got.Series[2].ActiveUsers)
	require.Equal(t, int64(700), got.Totals.TotalTokens)
	require.Equal(t, int64(3), got.Totals.NewUsers)
	require.Equal(t, int64(11), got.Totals.ActiveUsers)
}

func TestBuildEmptyDailyMetricsSeriesRejectsInvalidRange(t *testing.T) {
	require.Empty(t, buildEmptyDailyMetricsSeries("bad", "2026-03-01"))
	require.Empty(t, buildEmptyDailyMetricsSeries("2026-03-02", "2026-03-01"))
}
