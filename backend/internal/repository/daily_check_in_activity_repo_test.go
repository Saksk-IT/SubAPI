package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/DATA-DOG/go-sqlmock"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestDailyCheckInRepositoryClaimCreditsBalanceAtomically(t *testing.T) {
	repo, mock, closeDB := newDailyCheckInRepositoryMock(t)
	defer closeDB()
	checkedAt := time.Date(2026, 7, 19, 1, 2, 3, 0, time.UTC)

	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)SELECT enabled, reward_amount::float8.*FOR SHARE`).
		WillReturnRows(sqlmock.NewRows([]string{"enabled", "reward_amount"}).AddRow(true, 2.5))
	mock.ExpectQuery(`(?s)INSERT INTO daily_check_in_records.*AT TIME ZONE 'Asia/Shanghai'.*ON CONFLICT \(user_id, check_in_date\) DO NOTHING.*RETURNING id, check_in_date::text, created_at`).
		WithArgs(int64(42), 2.5).
		WillReturnRows(sqlmock.NewRows([]string{"id", "check_in_date", "created_at"}).AddRow(int64(7), "2026-07-19", checkedAt))
	mock.ExpectQuery(`(?s)UPDATE users\s+SET balance = balance \+ \$1, updated_at = NOW\(\)\s+WHERE id = \$2 AND deleted_at IS NULL\s+RETURNING balance::float8`).
		WithArgs(2.5, int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"balance"}).AddRow(12.5))
	mock.ExpectExec(`(?s)UPDATE daily_check_in_records\s+SET balance_after = \$1\s+WHERE id = \$2`).
		WithArgs(12.5, int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	claim, err := repo.Claim(context.Background(), 42)
	require.NoError(t, err)
	require.Equal(t, 2.5, claim.RewardAmount)
	require.Equal(t, 12.5, claim.BalanceAfter)
	require.Equal(t, "2026-07-19", claim.CheckInDate)
	require.Equal(t, checkedAt, claim.CheckedInAt)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDailyCheckInRepositoryClaimRejectsDuplicateDay(t *testing.T) {
	repo, mock, closeDB := newDailyCheckInRepositoryMock(t)
	defer closeDB()

	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)SELECT enabled, reward_amount::float8.*FOR SHARE`).
		WillReturnRows(sqlmock.NewRows([]string{"enabled", "reward_amount"}).AddRow(true, 1.0))
	mock.ExpectQuery(`(?s)INSERT INTO daily_check_in_records.*AT TIME ZONE 'Asia/Shanghai'.*ON CONFLICT \(user_id, check_in_date\) DO NOTHING.*RETURNING id, check_in_date::text, created_at`).
		WithArgs(int64(42), 1.0).
		WillReturnRows(sqlmock.NewRows([]string{"id", "check_in_date", "created_at"}))
	mock.ExpectRollback()

	_, err := repo.Claim(context.Background(), 42)
	require.Equal(t, "DAILY_CHECK_IN_ALREADY_DONE", infraerrors.Reason(err))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDailyCheckInRepositoryClaimRollsBackWhenBalanceCreditFails(t *testing.T) {
	repo, mock, closeDB := newDailyCheckInRepositoryMock(t)
	defer closeDB()
	checkedAt := time.Date(2026, 7, 19, 1, 2, 3, 0, time.UTC)

	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)SELECT enabled, reward_amount::float8.*FOR SHARE`).
		WillReturnRows(sqlmock.NewRows([]string{"enabled", "reward_amount"}).AddRow(true, 2.5))
	mock.ExpectQuery(`(?s)INSERT INTO daily_check_in_records.*AT TIME ZONE 'Asia/Shanghai'.*ON CONFLICT \(user_id, check_in_date\) DO NOTHING.*RETURNING id, check_in_date::text, created_at`).
		WithArgs(int64(42), 2.5).
		WillReturnRows(sqlmock.NewRows([]string{"id", "check_in_date", "created_at"}).AddRow(int64(7), "2026-07-19", checkedAt))
	mock.ExpectQuery(`(?s)UPDATE users\s+SET balance = balance \+ \$1, updated_at = NOW\(\)\s+WHERE id = \$2 AND deleted_at IS NULL\s+RETURNING balance::float8`).
		WithArgs(2.5, int64(42)).
		WillReturnError(errors.New("database write failed"))
	mock.ExpectRollback()

	claim, err := repo.Claim(context.Background(), 42)
	require.Nil(t, claim)
	require.ErrorContains(t, err, "credit daily check-in reward")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDailyCheckInRepositoryUsesDatabaseShanghaiCalendarDay(t *testing.T) {
	repo, mock, closeDB := newDailyCheckInRepositoryMock(t)
	defer closeDB()

	mock.ExpectQuery(`(?s)today\.check_in_date = \(CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Shanghai'\)::date`).
		WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"viewed_at", "checked_in_today", "total", "latest"}).
			AddRow(nil, true, int64(1), time.Date(2026, 7, 19, 1, 2, 3, 0, time.UTC)))

	state, err := repo.GetUserState(context.Background(), 42)
	require.NoError(t, err)
	require.True(t, state.CheckedInToday)
	require.Equal(t, int64(1), state.TotalCheckIns)
	require.NoError(t, mock.ExpectationsWereMet())
}

func newDailyCheckInRepositoryMock(t *testing.T) (*dailyCheckInRepository, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	driver := entsql.OpenDB(dialect.Postgres, db)
	client := dbent.NewClient(dbent.Driver(driver))
	return &dailyCheckInRepository{client: client, sql: db}, mock, func() {
		_ = client.Close()
	}
}

var _ service.DailyCheckInRepository = (*dailyCheckInRepository)(nil)
