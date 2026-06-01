package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type firstRechargeRepository struct {
	client *dbent.Client
	sql    sqlExecutor
}

func NewFirstRechargeRepository(client *dbent.Client, sqlDB *sql.DB) service.FirstRechargeRepository {
	return &firstRechargeRepository{client: client, sql: sqlDB}
}

func (r *firstRechargeRepository) GetConfig(ctx context.Context) (*service.FirstRechargeConfig, error) {
	exec := r.executor(ctx)
	if exec == nil {
		return nil, errors.New("first recharge sql executor is not configured")
	}
	cfg := service.FirstRechargeConfig{}
	err := scanSingleRow(ctx, exec, `
INSERT INTO first_recharge_activity_config (id, enabled, eligibility_scope, eligible_since)
VALUES (1, FALSE, 'new_users_after_enabled', NULL)
ON CONFLICT (id) DO UPDATE SET id = EXCLUDED.id
RETURNING enabled, eligibility_scope, eligible_since, created_at, updated_at`, nil,
		&cfg.Enabled,
		&cfg.EligibilityScope,
		&cfg.EligibleSince,
		&cfg.CreatedAt,
		&cfg.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (r *firstRechargeRepository) SaveConfig(ctx context.Context, config service.FirstRechargeConfig, offers []service.FirstRechargeOfferInput) (*service.FirstRechargeAdminConfig, error) {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	txCtx := dbent.NewTxContext(ctx, tx)
	exec := txAwareSQLExecutor(txCtx, r.sql, r.client)
	if exec == nil {
		return nil, errors.New("first recharge tx executor is not configured")
	}

	if _, err := exec.ExecContext(txCtx, `
INSERT INTO first_recharge_activity_config (id, enabled, eligibility_scope, eligible_since, updated_at)
VALUES (1, $1, $2, $3, NOW())
ON CONFLICT (id) DO UPDATE SET
	enabled = EXCLUDED.enabled,
	eligibility_scope = EXCLUDED.eligibility_scope,
	eligible_since = EXCLUDED.eligible_since,
	updated_at = NOW()`,
		config.Enabled,
		config.EligibilityScope,
		config.EligibleSince,
	); err != nil {
		return nil, fmt.Errorf("save first recharge config: %w", err)
	}

	keptIDs := make([]int64, 0, len(offers))
	for _, offer := range offers {
		if offer.ID > 0 {
			if _, err := exec.ExecContext(txCtx, `
UPDATE first_recharge_offers
SET name = $2,
	description = $3,
	price = $4,
	amount = $5,
	enabled = $6,
	sort_order = $7,
	updated_at = NOW()
WHERE id = $1`,
				offer.ID,
				offer.Name,
				offer.Description,
				offer.Price,
				offer.Amount,
				offer.Enabled,
				offer.SortOrder,
			); err != nil {
				return nil, fmt.Errorf("update first recharge offer: %w", err)
			}
			keptIDs = append(keptIDs, offer.ID)
			continue
		}

		var id int64
		if err := scanSingleRow(txCtx, exec, `
INSERT INTO first_recharge_offers (name, description, price, amount, enabled, sort_order)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id`, []any{
			offer.Name,
			offer.Description,
			offer.Price,
			offer.Amount,
			offer.Enabled,
			offer.SortOrder,
		}, &id); err != nil {
			return nil, fmt.Errorf("insert first recharge offer: %w", err)
		}
		keptIDs = append(keptIDs, id)
	}

	if len(keptIDs) == 0 {
		if _, err := exec.ExecContext(txCtx, `UPDATE first_recharge_offers SET enabled = FALSE, updated_at = NOW()`); err != nil {
			return nil, fmt.Errorf("disable removed first recharge offers: %w", err)
		}
	} else {
		if _, err := exec.ExecContext(txCtx, `
UPDATE first_recharge_offers
SET enabled = FALSE, updated_at = NOW()
WHERE id <> ALL($1)`, pq.Array(keptIDs)); err != nil {
			return nil, fmt.Errorf("disable removed first recharge offers: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit first recharge config: %w", err)
	}
	return r.loadAdminConfig(ctx)
}

func (r *firstRechargeRepository) ListOffers(ctx context.Context) ([]service.FirstRechargeOffer, error) {
	return r.listOffers(ctx, false)
}

func (r *firstRechargeRepository) ListEnabledOffers(ctx context.Context) ([]service.FirstRechargeOffer, error) {
	return r.listOffers(ctx, true)
}

func (r *firstRechargeRepository) GetEnabledOfferByID(ctx context.Context, offerID int64) (*service.FirstRechargeOffer, error) {
	exec := r.executor(ctx)
	if exec == nil {
		return nil, errors.New("first recharge sql executor is not configured")
	}
	rows, err := exec.QueryContext(ctx, `
SELECT id, name, description, price::float8, amount::float8, enabled, sort_order, created_at, updated_at
FROM first_recharge_offers
WHERE id = $1 AND enabled = TRUE`, offerID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, nil
	}
	offer, err := scanFirstRechargeOffer(rows)
	if err != nil {
		return nil, err
	}
	return &offer, rows.Err()
}

func (r *firstRechargeRepository) GetUserState(ctx context.Context, userID int64) (*service.FirstRechargeUserState, error) {
	exec := r.executor(ctx)
	if exec == nil {
		return nil, errors.New("first recharge sql executor is not configured")
	}
	state := service.FirstRechargeUserState{}
	err := scanSingleRow(ctx, exec, `
SELECT user_id, popup_dismissed_at, completed_order_id, completed_at, created_at, updated_at
FROM first_recharge_user_states
WHERE user_id = $1`, []any{userID},
		&state.UserID,
		&state.PopupDismissedAt,
		&state.CompletedOrderID,
		&state.CompletedAt,
		&state.CreatedAt,
		&state.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &state, nil
}

func (r *firstRechargeRepository) DismissPopup(ctx context.Context, userID int64, dismissedAt time.Time) error {
	exec := r.executor(ctx)
	if exec == nil {
		return errors.New("first recharge sql executor is not configured")
	}
	_, err := exec.ExecContext(ctx, `
INSERT INTO first_recharge_user_states (user_id, popup_dismissed_at, updated_at)
VALUES ($1, $2, NOW())
ON CONFLICT (user_id) DO UPDATE SET
	popup_dismissed_at = COALESCE(first_recharge_user_states.popup_dismissed_at, EXCLUDED.popup_dismissed_at),
	updated_at = NOW()`, userID, dismissedAt)
	return err
}

func (r *firstRechargeRepository) MarkCompleted(ctx context.Context, userID, orderID int64, completedAt time.Time) error {
	exec := r.executor(ctx)
	if exec == nil {
		return errors.New("first recharge sql executor is not configured")
	}
	_, err := exec.ExecContext(ctx, `
INSERT INTO first_recharge_user_states (user_id, completed_order_id, completed_at, updated_at)
VALUES ($1, $2, $3, NOW())
ON CONFLICT (user_id) DO UPDATE SET
	completed_order_id = CASE
		WHEN first_recharge_user_states.completed_at IS NULL THEN EXCLUDED.completed_order_id
		ELSE first_recharge_user_states.completed_order_id
	END,
	completed_at = COALESCE(first_recharge_user_states.completed_at, EXCLUDED.completed_at),
	updated_at = NOW()`, userID, orderID, completedAt)
	return err
}

func (r *firstRechargeRepository) HasCompleted(ctx context.Context, userID int64) (bool, error) {
	exec := r.executor(ctx)
	if exec == nil {
		return false, errors.New("first recharge sql executor is not configured")
	}
	var exists bool
	err := scanSingleRow(ctx, exec, `
SELECT EXISTS (
	SELECT 1 FROM first_recharge_user_states
	WHERE user_id = $1 AND completed_at IS NOT NULL
)`, []any{userID}, &exists)
	return exists, err
}

func (r *firstRechargeRepository) IsSpecifiedUser(ctx context.Context, userID int64) (bool, error) {
	exec := r.executor(ctx)
	if exec == nil {
		return false, errors.New("first recharge sql executor is not configured")
	}
	var exists bool
	err := scanSingleRow(ctx, exec, `
SELECT EXISTS (
	SELECT 1 FROM first_recharge_activity_users WHERE user_id = $1
)`, []any{userID}, &exists)
	return exists, err
}

func (r *firstRechargeRepository) ListSpecifiedUsers(ctx context.Context, params pagination.PaginationParams, search string) ([]service.FirstRechargeSpecifiedUser, *pagination.PaginationResult, error) {
	exec := r.executor(ctx)
	if exec == nil {
		return nil, nil, errors.New("first recharge sql executor is not configured")
	}

	where := ""
	args := []any{}
	if q := strings.TrimSpace(search); q != "" {
		args = append(args, "%"+q+"%")
		where = `WHERE u.email ILIKE $1 OR u.username ILIKE $1 OR u.id::text = trim(both '%' from $1)`
	}

	var total int64
	countQuery := `
SELECT COUNT(*)
FROM first_recharge_activity_users au
JOIN users u ON u.id = au.user_id ` + where
	if err := scanSingleRow(ctx, exec, countQuery, args, &total); err != nil {
		return nil, nil, err
	}

	dataArgs := append([]any{}, args...)
	dataArgs = append(dataArgs, params.Limit(), params.Offset())
	limitPos := len(dataArgs) - 1
	offsetPos := len(dataArgs)
	rows, err := exec.QueryContext(ctx, fmt.Sprintf(`
SELECT au.user_id, u.email, u.username, au.created_at
FROM first_recharge_activity_users au
JOIN users u ON u.id = au.user_id
%s
ORDER BY au.created_at DESC, au.user_id DESC
LIMIT $%d OFFSET $%d`, where, limitPos, offsetPos), dataArgs...)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = rows.Close() }()

	items := make([]service.FirstRechargeSpecifiedUser, 0)
	for rows.Next() {
		var item service.FirstRechargeSpecifiedUser
		if err := rows.Scan(&item.UserID, &item.Email, &item.Username, &item.CreatedAt); err != nil {
			return nil, nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	return items, paginationResultFromTotal(total, params), nil
}

func (r *firstRechargeRepository) AddSpecifiedUser(ctx context.Context, userID int64, actorID *int64) error {
	exec := r.executor(ctx)
	if exec == nil {
		return errors.New("first recharge sql executor is not configured")
	}
	_, err := exec.ExecContext(ctx, `
INSERT INTO first_recharge_activity_users (user_id, created_by)
VALUES ($1, $2)
ON CONFLICT (user_id) DO NOTHING`, userID, actorID)
	return err
}

func (r *firstRechargeRepository) RemoveSpecifiedUser(ctx context.Context, userID int64) error {
	exec := r.executor(ctx)
	if exec == nil {
		return errors.New("first recharge sql executor is not configured")
	}
	_, err := exec.ExecContext(ctx, `DELETE FROM first_recharge_activity_users WHERE user_id = $1`, userID)
	return err
}

func (r *firstRechargeRepository) loadAdminConfig(ctx context.Context) (*service.FirstRechargeAdminConfig, error) {
	cfg, err := r.GetConfig(ctx)
	if err != nil {
		return nil, err
	}
	offers, err := r.ListOffers(ctx)
	if err != nil {
		return nil, err
	}
	return &service.FirstRechargeAdminConfig{Config: *cfg, Offers: offers}, nil
}

func (r *firstRechargeRepository) listOffers(ctx context.Context, enabledOnly bool) ([]service.FirstRechargeOffer, error) {
	exec := r.executor(ctx)
	if exec == nil {
		return nil, errors.New("first recharge sql executor is not configured")
	}
	where := ""
	if enabledOnly {
		where = "WHERE enabled = TRUE"
	}
	rows, err := exec.QueryContext(ctx, `
SELECT id, name, description, price::float8, amount::float8, enabled, sort_order, created_at, updated_at
FROM first_recharge_offers
`+where+`
ORDER BY sort_order ASC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	offers := make([]service.FirstRechargeOffer, 0)
	for rows.Next() {
		offer, err := scanFirstRechargeOffer(rows)
		if err != nil {
			return nil, err
		}
		offers = append(offers, offer)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return offers, nil
}

func (r *firstRechargeRepository) executor(ctx context.Context) sqlQueryExecutor {
	return txAwareSQLExecutor(ctx, r.sql, r.client)
}

func scanFirstRechargeOffer(rows interface {
	Scan(dest ...any) error
}) (service.FirstRechargeOffer, error) {
	var offer service.FirstRechargeOffer
	err := rows.Scan(
		&offer.ID,
		&offer.Name,
		&offer.Description,
		&offer.Price,
		&offer.Amount,
		&offer.Enabled,
		&offer.SortOrder,
		&offer.CreatedAt,
		&offer.UpdatedAt,
	)
	return offer, err
}
