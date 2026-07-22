//go:build unit

package handler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type toggleSettingRepo struct {
	mu     sync.Mutex
	values map[string]string
}

func (r *toggleSettingRepo) Get(context.Context, string) (*service.Setting, error) { return nil, nil }
func (r *toggleSettingRepo) GetValue(_ context.Context, key string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.values[key], nil
}

func (r *toggleSettingRepo) Set(_ context.Context, key, value string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.values[key] = value
	return nil
}

func (r *toggleSettingRepo) GetMultiple(context.Context, []string) (map[string]string, error) {
	return map[string]string{}, nil
}
func (r *toggleSettingRepo) SetMultiple(context.Context, map[string]string) error { return nil }
func (r *toggleSettingRepo) GetAll(context.Context) (map[string]string, error) {
	return map[string]string{}, nil
}
func (r *toggleSettingRepo) Delete(context.Context, string) error { return nil }

type passthroughEncryptor struct{}

func (passthroughEncryptor) Encrypt(plaintext string) (string, error)  { return plaintext, nil }
func (passthroughEncryptor) Decrypt(ciphertext string) (string, error) { return ciphertext, nil }

type toggleImageStorage struct {
	saved chan []byte
}

func (s *toggleImageStorage) Save(_ context.Context, _ string, _ string, data []byte) (string, error) {
	s.saved <- append([]byte(nil), data...)
	return "https://cdn.example.test/object.png", nil
}

// TestAsyncImageEnablesWithoutRestart drives the actual HTTP path for the bug behind
// #4458 and #4542: with object storage unconfigured the async endpoint 404s, and the
// only way to turn it on used to be editing config.yaml and restarting the container.
// Flipping the admin setting must flip the endpoint over in the same process.
func TestAsyncImageEnablesWithoutRestart(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &toggleSettingRepo{values: map[string]string{}}
	// A fixed encryption key is required to persist a new S3 secret (#4524).
	backup := service.NewBackupService(repo, &config.Config{
		Totp: config.TotpConfig{EncryptionKeyConfigured: true},
	}, passthroughEncryptor{}, nil, nil)
	storage := &toggleImageStorage{saved: make(chan []byte, 1)}
	factory := func(context.Context, *config.ImageStorageConfig) (service.ImageStorage, error) {
		return storage, nil
	}
	settings := service.NewImageStorageSettingService(repo, passthroughEncryptor{}, backup, factory, config.ImageStorageConfig{})

	store := &asyncImageMemoryStore{tasks: make(map[string]*service.ImageTaskRecord)}
	tasks := service.NewImageTaskServiceWithResolver(store, settings.Resolver(), time.Hour, time.Minute)

	rawImage := []byte("\x89PNG\r\n\x1a\naccepted-storage-snapshot")
	b64Image := base64.StdEncoding.EncodeToString(rawImage)
	executionStarted := make(chan struct{})
	releaseExecution := make(chan struct{})
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(releaseExecution) }) }
	t.Cleanup(release)

	h := &AsyncImageHandler{tasks: tasks}
	h.execute = func(_ string, c *gin.Context) {
		close(executionStarted)
		<-releaseExecution
		c.JSON(http.StatusOK, gin.H{"created": 1, "data": []gin.H{{"b64_json": b64Image}}})
	}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		groupID := int64(3)
		c.Set(string(middleware2.ContextKeyAPIKey), &service.APIKey{
			ID: 9, UserID: 7, GroupID: &groupID,
			Group: &service.Group{ID: groupID, Platform: service.PlatformOpenAI, AllowImageGeneration: true},
		})
		c.Next()
	})
	router.POST("/v1/images/generations/async", h.Submit)
	router.GET("/v1/images/tasks/:task_id", h.Get)

	submit := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/v1/images/generations/async",
			strings.NewReader(`{"model":"gpt-image-1","prompt":"a lighthouse"}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}

	rec := submit()
	require.Equal(t, http.StatusNotFound, rec.Code, "disabled until an admin configures object storage")
	require.Contains(t, rec.Body.String(), "async image tasks are not enabled")

	// The admin saves the setting — no restart, same process.
	_, err := settings.Update(context.Background(), service.ImageStorageSettings{
		Enabled: true, Bucket: "my-images",
		Endpoint: "https://acct.r2.cloudflarestorage.com", AccessKeyID: "ak", SecretAccessKey: "sk",
	})
	require.NoError(t, err)

	rec = submit()
	require.Equal(t, http.StatusAccepted, rec.Code, "the endpoint must go live as soon as the setting is saved")

	var accepted struct {
		TaskID  string `json:"task_id"`
		PollURL string `json:"poll_url"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &accepted))
	require.NotEmpty(t, accepted.TaskID)
	select {
	case <-executionStarted:
	case <-time.After(time.Second):
		t.Fatal("accepted task did not start")
	}

	// Turning the feature back off must not strand a task that was already accepted
	// or make its raw base64 response fall back into Redis.
	_, err = settings.Update(context.Background(), service.ImageStorageSettings{Enabled: false})
	require.NoError(t, err)

	require.Equal(t, http.StatusNotFound, submit().Code, "new submissions are refused again")
	release()

	var completed *service.ImageTask
	require.Eventually(t, func() bool {
		var getErr error
		completed, getErr = tasks.Get(context.Background(), service.ImageTaskOwner{UserID: 7, APIKeyID: 9}, accepted.TaskID)
		return getErr == nil && completed.Status == service.ImageTaskStatusCompleted
	}, time.Second, 10*time.Millisecond)
	require.Equal(t, "https://cdn.example.test/object.png", completed.ImageURL)
	require.NotContains(t, string(completed.Result), "b64_json")
	require.NotContains(t, string(completed.Result), b64Image)
	select {
	case saved := <-storage.saved:
		require.Equal(t, rawImage, saved, "the accepted uploader must remain usable after settings are disabled")
	default:
		t.Fatal("accepted task did not offload its image")
	}

	pollRec := httptest.NewRecorder()
	router.ServeHTTP(pollRec, httptest.NewRequest(http.MethodGet, accepted.PollURL, nil))
	require.Equal(t, http.StatusOK, pollRec.Code, "an already-accepted task stays pollable after the switch is turned off")
}
