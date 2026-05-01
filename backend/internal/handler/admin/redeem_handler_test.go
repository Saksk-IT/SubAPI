package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newCreateAndRedeemHandler creates a RedeemHandler with a non-nil (but minimal)
// RedeemService so that CreateAndRedeem's nil guard passes and we can test the
// parameter-validation layer that runs before any service call.
func newCreateAndRedeemHandler() *RedeemHandler {
	return &RedeemHandler{
		adminService:  newStubAdminService(),
		redeemService: &service.RedeemService{}, // non-nil to pass nil guard
	}
}

// postCreateAndRedeemValidation calls CreateAndRedeem and returns the response
// status code. For cases that pass validation and proceed into the service layer,
// a panic may occur (because RedeemService internals are nil); this is expected
// and treated as "validation passed" (returns 0 to indicate panic).
func postCreateAndRedeemValidation(t *testing.T, handler *RedeemHandler, body any) (code int) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	jsonBytes, err := json.Marshal(body)
	require.NoError(t, err)
	c.Request, _ = http.NewRequest(http.MethodPost, "/api/v1/admin/redeem-codes/create-and-redeem", bytes.NewReader(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	defer func() {
		if r := recover(); r != nil {
			// Panic means we passed validation and entered service layer (expected for minimal stub).
			code = 0
		}
	}()
	handler.CreateAndRedeem(c)
	return w.Code
}

func TestCreateAndRedeem_TypeDefaultsToBalance(t *testing.T) {
	// 不传 type 字段时应默认 balance，不触发 subscription 校验。
	// 验证通过后进入 service 层会 panic（返回 0），说明默认值生效。
	h := newCreateAndRedeemHandler()
	code := postCreateAndRedeemValidation(t, h, map[string]any{
		"code":    "test-balance-default",
		"value":   10.0,
		"user_id": 1,
	})

	assert.NotEqual(t, http.StatusBadRequest, code,
		"omitting type should default to balance and pass validation")
}

func TestCreateAndRedeem_SubscriptionRequiresGroupID(t *testing.T) {
	h := newCreateAndRedeemHandler()
	code := postCreateAndRedeemValidation(t, h, map[string]any{
		"code":          "test-sub-no-group",
		"type":          "subscription",
		"value":         29.9,
		"user_id":       1,
		"validity_days": 30,
		// group_id 缺失
	})

	assert.Equal(t, http.StatusBadRequest, code)
}

func TestCreateAndRedeem_SubscriptionRequiresNonZeroValidityDays(t *testing.T) {
	groupID := int64(5)
	h := newCreateAndRedeemHandler()

	// zero should be rejected
	t.Run("zero", func(t *testing.T) {
		code := postCreateAndRedeemValidation(t, h, map[string]any{
			"code":          "test-sub-bad-days-zero",
			"type":          "subscription",
			"value":         29.9,
			"user_id":       1,
			"group_id":      groupID,
			"validity_days": 0,
		})

		assert.Equal(t, http.StatusBadRequest, code)
	})

	// negative should pass validation (used for refund/reduction)
	t.Run("negative_passes_validation", func(t *testing.T) {
		code := postCreateAndRedeemValidation(t, h, map[string]any{
			"code":          "test-sub-negative-days",
			"type":          "subscription",
			"value":         29.9,
			"user_id":       1,
			"group_id":      groupID,
			"validity_days": -7,
		})

		assert.NotEqual(t, http.StatusBadRequest, code,
			"negative validity_days should pass validation for refund")
	})
}

func TestCreateAndRedeem_SubscriptionValidParamsPassValidation(t *testing.T) {
	groupID := int64(5)
	h := newCreateAndRedeemHandler()
	code := postCreateAndRedeemValidation(t, h, map[string]any{
		"code":          "test-sub-valid",
		"type":          "subscription",
		"value":         29.9,
		"user_id":       1,
		"group_id":      groupID,
		"validity_days": 31,
	})

	assert.NotEqual(t, http.StatusBadRequest, code,
		"valid subscription params should pass validation")
}

func TestCreateAndRedeem_BalanceIgnoresSubscriptionFields(t *testing.T) {
	h := newCreateAndRedeemHandler()
	// balance 类型不传 group_id 和 validity_days，不应报 400
	code := postCreateAndRedeemValidation(t, h, map[string]any{
		"code":    "test-balance-no-extras",
		"type":    "balance",
		"value":   50.0,
		"user_id": 1,
	})

	assert.NotEqual(t, http.StatusBadRequest, code,
		"balance type should not require group_id or validity_days")
}

func setupRedeemManagementRouter() (*gin.Engine, *stubAdminService) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	adminSvc := newStubAdminService()
	h := NewRedeemHandler(adminSvc, nil)
	router.POST("/api/v1/admin/redeem-codes", h.Create)
	router.PUT("/api/v1/admin/redeem-codes/:id", h.Update)
	return router, adminSvc
}

func TestRedeemManagementCreatePassesPayload(t *testing.T) {
	router, adminSvc := setupRedeemManagementRouter()
	body, err := json.Marshal(map[string]any{
		"code":          " SUB-CODE ",
		"type":          "subscription",
		"value":         0,
		"group_id":      12,
		"validity_days": 45,
		"notes":         "partner generated",
	})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/redeem-codes", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, adminSvc.lastCreateRedeemCode)
	require.Equal(t, " SUB-CODE ", adminSvc.lastCreateRedeemCode.Code)
	require.Equal(t, service.RedeemTypeSubscription, adminSvc.lastCreateRedeemCode.Type)
	require.NotNil(t, adminSvc.lastCreateRedeemCode.GroupID)
	require.Equal(t, int64(12), *adminSvc.lastCreateRedeemCode.GroupID)
	require.Equal(t, 45, adminSvc.lastCreateRedeemCode.ValidityDays)
}

func TestRedeemManagementUpdatePassesPayload(t *testing.T) {
	router, adminSvc := setupRedeemManagementRouter()
	body, err := json.Marshal(map[string]any{
		"status":         "expired",
		"clear_group_id": true,
	})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/redeem-codes/5", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, int64(5), adminSvc.lastUpdateRedeemCode.id)
	require.NotNil(t, adminSvc.lastUpdateRedeemCode.input)
	require.NotNil(t, adminSvc.lastUpdateRedeemCode.input.Status)
	require.Equal(t, "expired", *adminSvc.lastUpdateRedeemCode.input.Status)
	require.True(t, adminSvc.lastUpdateRedeemCode.input.ClearGroupID)
}
