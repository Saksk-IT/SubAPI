package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func imageGenerationSettingsResponseData(t *testing.T, body []byte) map[string]any {
	t.Helper()

	var resp response.Response
	require.NoError(t, json.Unmarshal(body, &resp))
	data, ok := resp.Data.(map[string]any)
	require.True(t, ok)
	return data
}

func TestSettingHandler_GetSettings_ImageGenerationDefaultsEnabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &settingHandlerRepoStub{values: map[string]string{}}
	svc := service.NewSettingService(repo, &config.Config{Default: config.DefaultConfig{UserConcurrency: 5}})
	handler := NewSettingHandler(svc, nil, nil, nil, nil, nil, nil)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/settings", nil)
	handler.GetSettings(c)

	require.Equal(t, http.StatusOK, rec.Code)
	data := imageGenerationSettingsResponseData(t, rec.Body.Bytes())
	value, ok := data["image_generation_enabled"]
	require.True(t, ok, "admin settings are missing image_generation_enabled")
	require.Equal(t, true, value)
}

func TestSettingHandler_UpdateSettings_ImageGenerationRoundTrip(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &settingHandlerRepoStub{values: map[string]string{}}
	svc := service.NewSettingService(repo, &config.Config{Default: config.DefaultConfig{UserConcurrency: 5}})
	handler := NewSettingHandler(svc, nil, nil, nil, nil, nil, nil)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings", bytes.NewBufferString(`{"image_generation_enabled":false}`))
	c.Request.Header.Set("Content-Type", "application/json")
	handler.UpdateSettings(c)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "false", repo.values[service.SettingKeyImageGenerationEnabled])
	require.Equal(t, false, imageGenerationSettingsResponseData(t, rec.Body.Bytes())["image_generation_enabled"])

	getRec := httptest.NewRecorder()
	getContext, _ := gin.CreateTestContext(getRec)
	getContext.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/settings", nil)
	handler.GetSettings(getContext)

	require.Equal(t, http.StatusOK, getRec.Code)
	require.Equal(t, false, imageGenerationSettingsResponseData(t, getRec.Body.Bytes())["image_generation_enabled"])
}

func TestSettingHandler_UpdateSettings_PreservesOmittedImageGeneration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &settingHandlerRepoStub{values: map[string]string{
		service.SettingKeyImageGenerationEnabled: "false",
		service.SettingKeyPromoCodeEnabled:       "true",
	}}
	svc := service.NewSettingService(repo, &config.Config{Default: config.DefaultConfig{UserConcurrency: 5}})
	handler := NewSettingHandler(svc, nil, nil, nil, nil, nil, nil)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings", bytes.NewBufferString(`{"promo_code_enabled":true}`))
	c.Request.Header.Set("Content-Type", "application/json")
	handler.UpdateSettings(c)

	require.Equal(t, http.StatusOK, rec.Code)
	_, ok := repo.lastUpdates[service.SettingKeyImageGenerationEnabled]
	require.False(t, ok, "omitted image_generation_enabled must not be written")
	require.Equal(t, "false", repo.values[service.SettingKeyImageGenerationEnabled])
}

func TestDiffSettings_TracksImageGenerationEnabled(t *testing.T) {
	before := &service.SystemSettings{ImageGenerationEnabled: true}
	after := &service.SystemSettings{ImageGenerationEnabled: false}

	changed := diffSettings(before, after, nil, nil, UpdateSettingsRequest{})
	require.Equal(t, []string{"image_generation_enabled"}, changed)
}
