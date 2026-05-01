package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

const (
	maxGenerateRedeemCodes   = 1000
	defaultRedeemValidityDay = 30
)

type CreateRedeemCodeInput struct {
	Code         string
	Type         string
	Value        float64
	Status       string
	Notes        string
	GroupID      *int64
	ValidityDays int
}

type UpdateRedeemCodeInput struct {
	Code         *string
	Type         *string
	Value        *float64
	Status       *string
	Notes        *string
	GroupID      *int64
	ClearGroupID bool
	ValidityDays *int
}

type GenerateRedeemCodesInput struct {
	Count        int
	Type         string
	Value        float64
	GroupID      *int64 // 订阅类型专用：关联的分组ID
	ValidityDays int    // 订阅类型专用：有效天数
}

func (s *adminServiceImpl) ListRedeemCodes(ctx context.Context, page, pageSize int, codeType, status, search string, sortBy, sortOrder string) ([]RedeemCode, int64, error) {
	params := pagination.PaginationParams{Page: page, PageSize: pageSize, SortBy: sortBy, SortOrder: sortOrder}
	codes, result, err := s.redeemCodeRepo.ListWithFilters(ctx, params, codeType, status, search)
	if err != nil {
		return nil, 0, err
	}
	return codes, result.Total, nil
}

func (s *adminServiceImpl) GetRedeemCode(ctx context.Context, id int64) (*RedeemCode, error) {
	return s.redeemCodeRepo.GetByID(ctx, id)
}

func (s *adminServiceImpl) CreateRedeemCode(ctx context.Context, input *CreateRedeemCodeInput) (*RedeemCode, error) {
	if input == nil {
		return nil, infraerrors.BadRequest("INVALID_REDEEM_CODE", "redeem code input is required")
	}

	codeType, err := normalizeRedeemCodeType(input.Type)
	if err != nil {
		return nil, err
	}
	status, err := normalizeRedeemCodeStatus(input.Status)
	if err != nil {
		return nil, err
	}
	codeValue, err := normalizeRedeemCodeValue(input.Code, true)
	if err != nil {
		return nil, err
	}
	if codeValue == "" {
		codeValue, err = GenerateRedeemCode()
		if err != nil {
			return nil, fmt.Errorf("generate redeem code: %w", err)
		}
	}

	code := RedeemCode{
		Code:         codeValue,
		Type:         codeType,
		Value:        input.Value,
		Status:       status,
		Notes:        strings.TrimSpace(input.Notes),
		GroupID:      cloneInt64Ptr(input.GroupID),
		ValidityDays: input.ValidityDays,
	}
	if err := s.normalizeRedeemCodeBusinessFields(ctx, &code); err != nil {
		return nil, err
	}
	if err := s.redeemCodeRepo.Create(ctx, &code); err != nil {
		return nil, fmt.Errorf("create redeem code: %w", err)
	}
	created, err := s.redeemCodeRepo.GetByID(ctx, code.ID)
	if err != nil {
		return &code, nil
	}
	return created, nil
}

func (s *adminServiceImpl) UpdateRedeemCode(ctx context.Context, id int64, input *UpdateRedeemCodeInput) (*RedeemCode, error) {
	if input == nil {
		return nil, infraerrors.BadRequest("INVALID_REDEEM_CODE", "redeem code input is required")
	}

	existing, err := s.redeemCodeRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	updated := *existing

	if input.Code != nil {
		updated.Code, err = normalizeRedeemCodeValue(*input.Code, false)
		if err != nil {
			return nil, err
		}
	}
	if input.Type != nil {
		updated.Type, err = normalizeRedeemCodeType(*input.Type)
		if err != nil {
			return nil, err
		}
	}
	if input.Value != nil {
		updated.Value = *input.Value
	}
	if input.Status != nil {
		if existing.Status == StatusUsed {
			return nil, infraerrors.Conflict("REDEEM_CODE_USED", "used redeem code status cannot be changed")
		}
		updated.Status, err = normalizeRedeemCodeStatus(*input.Status)
		if err != nil {
			return nil, err
		}
	}
	if input.Notes != nil {
		updated.Notes = strings.TrimSpace(*input.Notes)
	}
	if input.ClearGroupID {
		updated.GroupID = nil
	} else if input.GroupID != nil {
		updated.GroupID = cloneInt64Ptr(input.GroupID)
	}
	if input.ValidityDays != nil {
		updated.ValidityDays = *input.ValidityDays
	}

	if err := s.normalizeRedeemCodeBusinessFields(ctx, &updated); err != nil {
		return nil, err
	}
	if err := s.redeemCodeRepo.Update(ctx, &updated); err != nil {
		return nil, fmt.Errorf("update redeem code: %w", err)
	}
	latest, err := s.redeemCodeRepo.GetByID(ctx, updated.ID)
	if err != nil {
		return &updated, nil
	}
	return latest, nil
}

func (s *adminServiceImpl) GenerateRedeemCodes(ctx context.Context, input *GenerateRedeemCodesInput) ([]RedeemCode, error) {
	if input == nil {
		return nil, infraerrors.BadRequest("INVALID_REDEEM_CODE", "generate input is required")
	}
	if input.Count <= 0 {
		return nil, infraerrors.BadRequest("INVALID_REDEEM_CODE_COUNT", "count must be greater than 0")
	}
	if input.Count > maxGenerateRedeemCodes {
		return nil, infraerrors.BadRequest("INVALID_REDEEM_CODE_COUNT", fmt.Sprintf("count cannot exceed %d", maxGenerateRedeemCodes))
	}

	codeType, err := normalizeRedeemCodeType(input.Type)
	if err != nil {
		return nil, err
	}
	template := RedeemCode{
		Type:         codeType,
		Value:        input.Value,
		Status:       StatusUnused,
		GroupID:      cloneInt64Ptr(input.GroupID),
		ValidityDays: input.ValidityDays,
	}
	if err := s.normalizeRedeemCodeBusinessFields(ctx, &template); err != nil {
		return nil, err
	}

	codes := make([]RedeemCode, 0, input.Count)
	for i := 0; i < input.Count; i++ {
		codeValue, err := GenerateRedeemCode()
		if err != nil {
			return nil, fmt.Errorf("generate redeem code: %w", err)
		}
		code := template
		code.Code = codeValue
		if err := s.redeemCodeRepo.Create(ctx, &code); err != nil {
			return nil, fmt.Errorf("create redeem code: %w", err)
		}
		codes = append(codes, code)
	}
	return codes, nil
}

func (s *adminServiceImpl) DeleteRedeemCode(ctx context.Context, id int64) error {
	return s.redeemCodeRepo.Delete(ctx, id)
}

func (s *adminServiceImpl) BatchDeleteRedeemCodes(ctx context.Context, ids []int64) (int64, error) {
	var deleted int64
	for _, id := range ids {
		if err := s.redeemCodeRepo.Delete(ctx, id); err == nil {
			deleted++
		}
	}
	return deleted, nil
}

func (s *adminServiceImpl) ExpireRedeemCode(ctx context.Context, id int64) (*RedeemCode, error) {
	code, err := s.redeemCodeRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	updated := *code
	updated.Status = StatusExpired
	if err := s.redeemCodeRepo.Update(ctx, &updated); err != nil {
		return nil, err
	}
	return &updated, nil
}

func normalizeRedeemCodeType(raw string) (string, error) {
	codeType := strings.ToLower(strings.TrimSpace(raw))
	if codeType == "" {
		return RedeemTypeBalance, nil
	}
	switch codeType {
	case RedeemTypeBalance, RedeemTypeConcurrency, RedeemTypeSubscription, RedeemTypeInvitation:
		return codeType, nil
	default:
		return "", infraerrors.BadRequest("INVALID_REDEEM_CODE_TYPE", "type must be balance, concurrency, subscription, or invitation")
	}
}

func normalizeRedeemCodeStatus(raw string) (string, error) {
	status := strings.ToLower(strings.TrimSpace(raw))
	if status == "" {
		return StatusUnused, nil
	}
	switch status {
	case StatusUnused, StatusExpired:
		return status, nil
	case StatusUsed:
		return "", infraerrors.BadRequest("INVALID_REDEEM_CODE_STATUS", "status cannot be set to used directly")
	default:
		return "", infraerrors.BadRequest("INVALID_REDEEM_CODE_STATUS", "status must be unused or expired")
	}
}

func normalizeRedeemCodeValue(raw string, allowEmpty bool) (string, error) {
	code := strings.TrimSpace(raw)
	if code == "" {
		if allowEmpty {
			return "", nil
		}
		return "", infraerrors.BadRequest("INVALID_REDEEM_CODE", "code is required")
	}
	if len(code) > 32 {
		return "", infraerrors.BadRequest("INVALID_REDEEM_CODE", "code length cannot exceed 32")
	}
	return code, nil
}

func (s *adminServiceImpl) normalizeRedeemCodeBusinessFields(ctx context.Context, code *RedeemCode) error {
	if code == nil {
		return infraerrors.BadRequest("INVALID_REDEEM_CODE", "redeem code is required")
	}

	switch code.Type {
	case RedeemTypeBalance, RedeemTypeConcurrency:
		if code.Value == 0 {
			return infraerrors.BadRequest("INVALID_REDEEM_CODE_VALUE", "value must not be zero for balance or concurrency type")
		}
		code.GroupID = nil
		code.ValidityDays = 0
	case RedeemTypeInvitation:
		code.Value = 0
		code.GroupID = nil
		code.ValidityDays = 0
	case RedeemTypeSubscription:
		if code.GroupID == nil || *code.GroupID <= 0 {
			return infraerrors.BadRequest("INVALID_REDEEM_CODE_GROUP", "group_id is required for subscription type")
		}
		if code.ValidityDays < 0 {
			return infraerrors.BadRequest("INVALID_REDEEM_CODE_VALIDITY", "validity_days must be greater than 0")
		}
		if code.ValidityDays == 0 {
			code.ValidityDays = defaultRedeemValidityDay
		}
		if err := s.ensureSubscriptionGroup(ctx, *code.GroupID); err != nil {
			return err
		}
	default:
		return infraerrors.BadRequest("INVALID_REDEEM_CODE_TYPE", "unsupported redeem code type")
	}
	return nil
}

func (s *adminServiceImpl) ensureSubscriptionGroup(ctx context.Context, groupID int64) error {
	if s.groupRepo == nil {
		return infraerrors.InternalServer("GROUP_REPOSITORY_UNAVAILABLE", "group repository is unavailable")
	}
	group, err := s.groupRepo.GetByID(ctx, groupID)
	if err != nil {
		if errors.Is(err, ErrGroupNotFound) {
			return err
		}
		return fmt.Errorf("get group: %w", err)
	}
	if group == nil || !group.IsSubscriptionType() {
		return infraerrors.BadRequest("INVALID_REDEEM_CODE_GROUP", "group must be subscription type")
	}
	return nil
}

func cloneInt64Ptr(v *int64) *int64 {
	if v == nil {
		return nil
	}
	out := *v
	return &out
}
