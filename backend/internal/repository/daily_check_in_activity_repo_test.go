package repository

import (
	"context"
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
	mock.ExpectQuery(`(?s)INSERT INTO daily_check_in_records.*ON CONFLICT \(user_id, check_in_date\) DO NOTHING.*RETURNING id, created_at`).
		WithArgs(int64(42), "2026-07-19", 2.5).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow(int64(7), checkedAt))
	mock.ExpectQuery(`(?s)UPDATE users\s+SET balance = balance \+ \$1, updated_at = NOW\(\)\s+WHERE id = \$2 AND deleted_at IS NULL\s+RETURNING balance::float8`).
		WithArgs(2.5, int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"balance"}).AddRow(12.5))
	mock.ExpectExec(`(?s)UPDATE daily_check_in_records\s+SET balance_after = \$1\s+WHERE id = \$2`).
		WithArgs(12.5, int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	claim, err := repo.Claim(context.Background(), 42, "2026-07-19")
	require.NoError(t, err)
	require.Equal(t, 2.5, claim.RewardAmount)
	require.Equal(t, 12.5, claim.BalanceAfter)
	require.Equal(t, checkedAt, claim.CheckedInAt)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDailyCheckInRepositoryClaimRejectsDuplicateDay(t *testing.T) {
	repo, mock, closeDB := newDailyCheckInRepositoryMock(t)
	defer closeDB()

	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)SELECT enabled, reward_amount::float8.*FOR SHARE`).
		WillReturnRows(sqlmock.NewRows([]string{"enabled", "reward_amount"}).AddRow(true, 1.0))
	mock.ExpectQuery(`(?s)INSERT INTO daily_check_in_records.*ON CONFLICT \(user_id, check_in_date\) DO NOTHING.*RETURNING id, created_at`).
		WithArgs(int64(42), "2026-07-19", 1.0).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}))
	mock.ExpectRollback()

	_, err := repo.Claim(context.Background(), 42, "2026-07-19")
	require.Equal(t, "DAILY_CHECK_IN_ALREADY_DONE", infraerrors.Reason(err))
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
