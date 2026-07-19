package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type dailyCheckInRepository struct {
	client *dbent.Client
	sql    sqlExecutor
}

func NewDailyCheckInRepository(client *dbent.Client, sqlDB *sql.DB) service.DailyCheckInRepository {
	return &dailyCheckInRepository{client: client, sql: sqlDB}
}

func (r *dailyCheckInRepository) GetConfig(ctx context.Context) (*service.DailyCheckInConfig, error) {
	exec := r.executor(ctx)
	if exec == nil {
		return nil, errors.New("daily check-in sql executor is not configured")
	}
	if _, err := exec.ExecContext(ctx, `
INSERT INTO daily_check_in_activity_config (id, enabled, reward_amount)
VALUES (1, FALSE, 1)
ON CONFLICT (id) DO NOTHING`); err != nil {
		return nil, fmt.Errorf("ensure daily check-in config: %w", err)
	}
	config := &service.DailyCheckInConfig{Timezone: service.DailyCheckInTimezone}
	err := scanSingleRow(ctx, exec, `
SELECT enabled, reward_amount::float8, created_at, updated_at
FROM daily_check_in_activity_config
WHERE id = 1`, nil,
		&config.Enabled,
		&config.RewardAmount,
		&config.CreatedAt,
		&config.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return config, nil
}

func (r *dailyCheckInRepository) SaveConfig(ctx context.Context, enabled bool, rewardAmount float64) (*service.DailyCheckInConfig, error) {
	exec := r.executor(ctx)
	if exec == nil {
		return nil, errors.New("daily check-in sql executor is not configured")
	}
	config := &service.DailyCheckInConfig{Timezone: service.DailyCheckInTimezone}
	err := scanSingleRow(ctx, exec, `
INSERT INTO daily_check_in_activity_config (id, enabled, reward_amount, updated_at)
VALUES (1, $1, $2, NOW())
ON CONFLICT (id) DO UPDATE SET
	enabled = EXCLUDED.enabled,
	reward_amount = EXCLUDED.reward_amount,
	updated_at = NOW()
RETURNING enabled, reward_amount::float8, created_at, updated_at`, []any{enabled, rewardAmount},
		&config.Enabled,
		&config.RewardAmount,
		&config.CreatedAt,
		&config.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("save daily check-in config: %w", err)
	}
	return config, nil
}

func (r *dailyCheckInRepository) GetUserState(ctx context.Context, userID int64, checkInDate string) (*service.DailyCheckInUserState, error) {
	exec := r.executor(ctx)
	if exec == nil {
		return nil, errors.New("daily check-in sql executor is not configured")
	}
	state := &service.DailyCheckInUserState{}
	err := scanSingleRow(ctx, exec, `
SELECT s.viewed_at,
	EXISTS (
		SELECT 1 FROM daily_check_in_records today
		WHERE today.user_id = $1 AND today.check_in_date = $2::date
	),
	(SELECT COUNT(*) FROM daily_check_in_records history WHERE history.user_id = $1),
	(SELECT latest.created_at FROM daily_check_in_records latest
	 WHERE latest.user_id = $1 ORDER BY latest.created_at DESC, latest.id DESC LIMIT 1)
FROM (SELECT $1::bigint AS user_id) requested
LEFT JOIN daily_check_in_user_states s ON s.user_id = requested.user_id`, []any{userID, checkInDate},
		&state.ViewedAt,
		&state.CheckedInToday,
		&state.TotalCheckIns,
		&state.LastCheckedInAt,
	)
	if err != nil {
		return nil, err
	}
	return state, nil
}

func (r *dailyCheckInRepository) MarkViewed(ctx context.Context, userID int64, viewedAt time.Time) error {
	exec := r.executor(ctx)
	if exec == nil {
		return errors.New("daily check-in sql executor is not configured")
	}
	_, err := exec.ExecContext(ctx, `
INSERT INTO daily_check_in_user_states (user_id, viewed_at, updated_at)
VALUES ($1, $2, NOW())
ON CONFLICT (user_id) DO UPDATE SET
	viewed_at = COALESCE(daily_check_in_user_states.viewed_at, EXCLUDED.viewed_at),
	updated_at = NOW()`, userID, viewedAt)
	return err
}

func (r *dailyCheckInRepository) Claim(ctx context.Context, userID int64, checkInDate string) (*service.DailyCheckInClaim, error) {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin daily check-in transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	txCtx := dbent.NewTxContext(ctx, tx)
	exec := txAwareSQLExecutor(txCtx, r.sql, r.client)
	if exec == nil {
		return nil, errors.New("daily check-in tx executor is not configured")
	}

	var enabled bool
	var rewardAmount float64
	if err := scanSingleRow(txCtx, exec, `
SELECT enabled, reward_amount::float8
FROM daily_check_in_activity_config
WHERE id = 1
FOR SHARE`, nil, &enabled, &rewardAmount); err != nil {
		return nil, fmt.Errorf("load daily check-in config for claim: %w", err)
	}
	if !enabled {
		return nil, service.ErrDailyCheckInUnavailable
	}

	var recordID int64
	var checkedInAt time.Time
	err = scanSingleRow(txCtx, exec, `
INSERT INTO daily_check_in_records (user_id, check_in_date, reward_amount, balance_after)
VALUES ($1, $2::date, $3, 0)
ON CONFLICT (user_id, check_in_date) DO NOTHING
RETURNING id, created_at`, []any{userID, checkInDate, rewardAmount}, &recordID, &checkedInAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrDailyCheckInAlreadyDone
	}
	if err != nil {
		return nil, fmt.Errorf("create daily check-in record: %w", err)
	}

	var balanceAfter float64
	if err := scanSingleRow(txCtx, exec, `
UPDATE users
SET balance = balance + $1, updated_at = NOW()
WHERE id = $2 AND deleted_at IS NULL
RETURNING balance::float8`, []any{rewardAmount, userID}, &balanceAfter); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrUserNotFound
		}
		return nil, fmt.Errorf("credit daily check-in reward: %w", err)
	}

	if _, err := exec.ExecContext(txCtx, `
UPDATE daily_check_in_records
SET balance_after = $1
WHERE id = $2`, balanceAfter, recordID); err != nil {
		return nil, fmt.Errorf("save daily check-in balance snapshot: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit daily check-in transaction: %w", err)
	}
	return &service.DailyCheckInClaim{
		RewardAmount: rewardAmount,
		BalanceAfter: balanceAfter,
		CheckInDate:  checkInDate,
		CheckedInAt:  checkedInAt,
	}, nil
}

func (r *dailyCheckInRepository) executor(ctx context.Context) sqlQueryExecutor {
	return txAwareSQLExecutor(ctx, r.sql, r.client)
}
