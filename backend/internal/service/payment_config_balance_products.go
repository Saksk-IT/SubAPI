package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	maxProductTags     = 8
	maxProductTagLen   = 24
	maxProductFeatures = 12
)

// BalanceProduct is an admin-configured recharge product.
type BalanceProduct struct {
	ID            int64      `json:"id"`
	Name          string     `json:"name"`
	Description   string     `json:"description"`
	Price         float64    `json:"price"`
	Amount        float64    `json:"amount"`
	OriginalPrice *float64   `json:"original_price,omitempty"`
	Tags          string     `json:"tags"`
	Features      string     `json:"features"`
	ProductName   string     `json:"product_name"`
	ForSale       bool       `json:"for_sale"`
	SortOrder     int        `json:"sort_order"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type CreateBalanceProductRequest struct {
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	Price         float64  `json:"price"`
	Amount        float64  `json:"amount"`
	OriginalPrice *float64 `json:"original_price"`
	Tags          string   `json:"tags"`
	Features      string   `json:"features"`
	ProductName   string   `json:"product_name"`
	ForSale       bool     `json:"for_sale"`
	SortOrder     int      `json:"sort_order"`
}

type UpdateBalanceProductRequest struct {
	Name          *string  `json:"name"`
	Description   *string  `json:"description"`
	Price         *float64 `json:"price"`
	Amount        *float64 `json:"amount"`
	OriginalPrice *float64 `json:"original_price"`
	Tags          *string  `json:"tags"`
	Features      *string  `json:"features"`
	ProductName   *string  `json:"product_name"`
	ForSale       *bool    `json:"for_sale"`
	SortOrder     *int     `json:"sort_order"`
}

func validateBalanceProductRequired(req CreateBalanceProductRequest) error {
	if strings.TrimSpace(req.Name) == "" {
		return infraerrors.BadRequest("BALANCE_PRODUCT_NAME_REQUIRED", "product name is required")
	}
	if req.Price <= 0 {
		return infraerrors.BadRequest("BALANCE_PRODUCT_PRICE_INVALID", "price must be > 0")
	}
	if req.Amount <= 0 {
		return infraerrors.BadRequest("BALANCE_PRODUCT_AMOUNT_INVALID", "amount must be > 0")
	}
	if req.OriginalPrice != nil && *req.OriginalPrice < 0 {
		return infraerrors.BadRequest("BALANCE_PRODUCT_ORIGINAL_PRICE_INVALID", "original price must be >= 0")
	}
	if err := validateProductLines(req.Tags, maxProductTags, maxProductTagLen, "BALANCE_PRODUCT_TAGS_INVALID"); err != nil {
		return err
	}
	if err := validateProductLines(req.Features, maxProductFeatures, 160, "BALANCE_PRODUCT_FEATURES_INVALID"); err != nil {
		return err
	}
	return nil
}

func validateBalanceProductPatch(req UpdateBalanceProductRequest) error {
	if req.Name != nil && strings.TrimSpace(*req.Name) == "" {
		return infraerrors.BadRequest("BALANCE_PRODUCT_NAME_REQUIRED", "product name is required")
	}
	if req.Price != nil && *req.Price <= 0 {
		return infraerrors.BadRequest("BALANCE_PRODUCT_PRICE_INVALID", "price must be > 0")
	}
	if req.Amount != nil && *req.Amount <= 0 {
		return infraerrors.BadRequest("BALANCE_PRODUCT_AMOUNT_INVALID", "amount must be > 0")
	}
	if req.OriginalPrice != nil && *req.OriginalPrice < 0 {
		return infraerrors.BadRequest("BALANCE_PRODUCT_ORIGINAL_PRICE_INVALID", "original price must be >= 0")
	}
	if req.Tags != nil {
		if err := validateProductLines(*req.Tags, maxProductTags, maxProductTagLen, "BALANCE_PRODUCT_TAGS_INVALID"); err != nil {
			return err
		}
	}
	if req.Features != nil {
		if err := validateProductLines(*req.Features, maxProductFeatures, 160, "BALANCE_PRODUCT_FEATURES_INVALID"); err != nil {
			return err
		}
	}
	return nil
}

func validateProductLines(raw string, maxLines int, maxLen int, code string) error {
	lines := splitProductLines(raw)
	if len(lines) > maxLines {
		return infraerrors.BadRequest(code, fmt.Sprintf("too many entries, maximum %d", maxLines))
	}
	for _, line := range lines {
		if len([]rune(line)) > maxLen {
			return infraerrors.BadRequest(code, fmt.Sprintf("entry is too long, maximum %d characters", maxLen))
		}
	}
	return nil
}

func splitProductLines(raw string) []string {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	var out []string
	for _, line := range strings.Split(raw, "\n") {
		if s := strings.TrimSpace(line); s != "" {
			out = append(out, s)
		}
	}
	if out == nil {
		return []string{}
	}
	return out
}

func (s *PaymentConfigService) ListBalanceProducts(ctx context.Context) ([]*BalanceProduct, error) {
	rows, err := s.entClient.QueryContext(ctx, `
SELECT id, name, description, price, amount, original_price, tags, features, product_name, for_sale, sort_order, created_at, updated_at
FROM balance_products
ORDER BY sort_order ASC, id ASC`)
	if err != nil {
		return nil, fmt.Errorf("list balance products: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var products []*BalanceProduct
	for rows.Next() {
		product, err := scanBalanceProduct(rows)
		if err != nil {
			return nil, err
		}
		products = append(products, product)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate balance products: %w", err)
	}
	return products, nil
}

func (s *PaymentConfigService) ListBalanceProductsForSale(ctx context.Context) ([]*BalanceProduct, error) {
	rows, err := s.entClient.QueryContext(ctx, `
SELECT id, name, description, price, amount, original_price, tags, features, product_name, for_sale, sort_order, created_at, updated_at
FROM balance_products
WHERE for_sale = TRUE
ORDER BY sort_order ASC, id ASC`)
	if err != nil {
		return nil, fmt.Errorf("list sale balance products: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var products []*BalanceProduct
	for rows.Next() {
		product, err := scanBalanceProduct(rows)
		if err != nil {
			return nil, err
		}
		products = append(products, product)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sale balance products: %w", err)
	}
	return products, nil
}

func (s *PaymentConfigService) GetBalanceProduct(ctx context.Context, id int64) (*BalanceProduct, error) {
	rows, err := s.entClient.QueryContext(ctx, `
SELECT id, name, description, price, amount, original_price, tags, features, product_name, for_sale, sort_order, created_at, updated_at
FROM balance_products
WHERE id = $1`, id)
	if err != nil {
		return nil, fmt.Errorf("get balance product: %w", err)
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		return nil, infraerrors.NotFound("BALANCE_PRODUCT_NOT_FOUND", "balance product not found")
	}
	product, err := scanBalanceProduct(rows)
	if err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read balance product: %w", err)
	}
	return product, nil
}

func (s *PaymentConfigService) CreateBalanceProduct(ctx context.Context, req CreateBalanceProductRequest) (*BalanceProduct, error) {
	if err := validateBalanceProductRequired(req); err != nil {
		return nil, err
	}
	now := time.Now()
	var originalPrice any
	if req.OriginalPrice != nil {
		originalPrice = *req.OriginalPrice
	}
	rows, err := s.entClient.QueryContext(ctx, `
INSERT INTO balance_products (name, description, price, amount, original_price, tags, features, product_name, for_sale, sort_order, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING id, name, description, price, amount, original_price, tags, features, product_name, for_sale, sort_order, created_at, updated_at`,
		strings.TrimSpace(req.Name), strings.TrimSpace(req.Description), req.Price, req.Amount, originalPrice,
		normalizeProductLines(req.Tags), normalizeProductLines(req.Features), strings.TrimSpace(req.ProductName), req.ForSale, req.SortOrder, now, now)
	if err != nil {
		return nil, fmt.Errorf("create balance product: %w", err)
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		return nil, errors.New("create balance product returned no row")
	}
	return scanBalanceProduct(rows)
}

func (s *PaymentConfigService) UpdateBalanceProduct(ctx context.Context, id int64, req UpdateBalanceProductRequest) (*BalanceProduct, error) {
	if err := validateBalanceProductPatch(req); err != nil {
		return nil, err
	}
	current, err := s.GetBalanceProduct(ctx, id)
	if err != nil {
		return nil, err
	}
	next := *current
	if req.Name != nil {
		next.Name = strings.TrimSpace(*req.Name)
	}
	if req.Description != nil {
		next.Description = strings.TrimSpace(*req.Description)
	}
	if req.Price != nil {
		next.Price = *req.Price
	}
	if req.Amount != nil {
		next.Amount = *req.Amount
	}
	if req.OriginalPrice != nil {
		next.OriginalPrice = req.OriginalPrice
	}
	if req.Tags != nil {
		next.Tags = normalizeProductLines(*req.Tags)
	}
	if req.Features != nil {
		next.Features = normalizeProductLines(*req.Features)
	}
	if req.ProductName != nil {
		next.ProductName = strings.TrimSpace(*req.ProductName)
	}
	if req.ForSale != nil {
		next.ForSale = *req.ForSale
	}
	if req.SortOrder != nil {
		next.SortOrder = *req.SortOrder
	}

	var originalPrice any
	if next.OriginalPrice != nil {
		originalPrice = *next.OriginalPrice
	}
	rows, err := s.entClient.QueryContext(ctx, `
UPDATE balance_products
SET name = $2, description = $3, price = $4, amount = $5, original_price = $6, tags = $7, features = $8,
    product_name = $9, for_sale = $10, sort_order = $11, updated_at = $12
WHERE id = $1
RETURNING id, name, description, price, amount, original_price, tags, features, product_name, for_sale, sort_order, created_at, updated_at`,
		id, next.Name, next.Description, next.Price, next.Amount, originalPrice, next.Tags, next.Features,
		next.ProductName, next.ForSale, next.SortOrder, time.Now())
	if err != nil {
		return nil, fmt.Errorf("update balance product: %w", err)
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		return nil, infraerrors.NotFound("BALANCE_PRODUCT_NOT_FOUND", "balance product not found")
	}
	return scanBalanceProduct(rows)
}

func (s *PaymentConfigService) DeleteBalanceProduct(ctx context.Context, id int64) error {
	res, err := s.entClient.ExecContext(ctx, `DELETE FROM balance_products WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete balance product: %w", err)
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return infraerrors.NotFound("BALANCE_PRODUCT_NOT_FOUND", "balance product not found")
	}
	return nil
}

type balanceProductScanner interface {
	Scan(dest ...any) error
}

func scanBalanceProduct(scanner balanceProductScanner) (*BalanceProduct, error) {
	var product BalanceProduct
	var original sql.NullFloat64
	if err := scanner.Scan(
		&product.ID,
		&product.Name,
		&product.Description,
		&product.Price,
		&product.Amount,
		&original,
		&product.Tags,
		&product.Features,
		&product.ProductName,
		&product.ForSale,
		&product.SortOrder,
		&product.CreatedAt,
		&product.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("scan balance product: %w", err)
	}
	if original.Valid {
		product.OriginalPrice = &original.Float64
	}
	return &product, nil
}

func normalizeProductLines(raw string) string {
	return strings.Join(splitProductLines(raw), "\n")
}
