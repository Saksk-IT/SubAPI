package service

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestParseOpenAIImagesMultipartRejectsOversizedPart(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("model", "gpt-image-2"))
	part, err := writer.CreateFormFile("image", "oversized.png")
	require.NoError(t, err)
	_, err = part.Write(bytes.Repeat([]byte{0x42}, openAIImageMaxUploadPartSize+1))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	_, err = parseOpenAIImagesForValidation(t, "/v1/images/edits", writer.FormDataContentType(), body.Bytes())
	require.ErrorContains(t, err, "exceeds 20MB limit")
}

func TestParseOpenAIImagesRequestRejectsParametersAboveLimits(t *testing.T) {
	tests := []struct {
		name      string
		field     string
		value     string
		wantError string
	}{
		{name: "output count", field: "n", value: "11", wantError: "n must be between 1 and 10"},
		{name: "partial images", field: "partial_images", value: "4", wantError: "partial_images must be between 0 and 3"},
		{name: "output compression", field: "output_compression", value: "101", wantError: "output_compression must be between 0 and 100"},
	}

	for _, test := range tests {
		t.Run("json "+test.name, func(t *testing.T) {
			body := []byte(fmt.Sprintf(`{"model":"gpt-image-2","prompt":"cat","%s":%s}`, test.field, test.value))
			_, err := parseOpenAIImagesForValidation(t, "/v1/images/generations", "application/json", body)
			require.ErrorContains(t, err, test.wantError)
		})

		t.Run("multipart "+test.name, func(t *testing.T) {
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			require.NoError(t, writer.WriteField("model", "gpt-image-2"))
			require.NoError(t, writer.WriteField("prompt", "cat"))
			require.NoError(t, writer.WriteField(test.field, test.value))
			require.NoError(t, writer.Close())

			_, err := parseOpenAIImagesForValidation(t, "/v1/images/generations", writer.FormDataContentType(), body.Bytes())
			require.ErrorContains(t, err, test.wantError)
		})
	}
}

func TestParseOpenAIImagesJSONRequestRejectsFractionalIntegerFields(t *testing.T) {
	tests := []struct {
		field string
		value string
	}{
		{field: "n", value: "10.9"},
		{field: "partial_images", value: "3.9"},
		{field: "output_compression", value: "100.9"},
	}

	for _, test := range tests {
		t.Run(test.field, func(t *testing.T) {
			body := []byte(fmt.Sprintf(`{"model":"gpt-image-2","prompt":"cat","%s":%s}`, test.field, test.value))
			_, err := parseOpenAIImagesForValidation(t, "/v1/images/generations", "application/json", body)
			require.ErrorContains(t, err, fmt.Sprintf("invalid %s field type", test.field))
		})
	}
}

func TestParseOpenAIImagesRequestRejectsParametersBelowLimits(t *testing.T) {
	tests := []struct {
		name      string
		field     string
		wantError string
	}{
		{name: "partial images", field: "partial_images", wantError: "partial_images must be between 0 and 3"},
		{name: "output compression", field: "output_compression", wantError: "output_compression must be between 0 and 100"},
	}

	for _, test := range tests {
		t.Run("json "+test.name, func(t *testing.T) {
			body := []byte(fmt.Sprintf(`{"model":"gpt-image-2","prompt":"cat","%s":-1}`, test.field))
			_, err := parseOpenAIImagesForValidation(t, "/v1/images/generations", "application/json", body)
			require.ErrorContains(t, err, test.wantError)
		})

		t.Run("multipart "+test.name, func(t *testing.T) {
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			require.NoError(t, writer.WriteField("model", "gpt-image-2"))
			require.NoError(t, writer.WriteField("prompt", "cat"))
			require.NoError(t, writer.WriteField(test.field, "-1"))
			require.NoError(t, writer.Close())

			_, err := parseOpenAIImagesForValidation(t, "/v1/images/generations", writer.FormDataContentType(), body.Bytes())
			require.ErrorContains(t, err, test.wantError)
		})
	}
}

func TestParseOpenAIImagesRequestRejectsMoreThanSixteenReferenceImages(t *testing.T) {
	t.Run("json image URLs", func(t *testing.T) {
		images := make([]string, 0, 17)
		for i := 0; i < 17; i++ {
			images = append(images, fmt.Sprintf(`{"image_url":"https://example.com/%d.png"}`, i))
		}
		body := []byte(fmt.Sprintf(`{"model":"gpt-image-2","prompt":"cat","images":[%s]}`, strings.Join(images, ",")))

		_, err := parseOpenAIImagesForValidation(t, "/v1/images/edits", "application/json", body)
		require.ErrorContains(t, err, "images must contain at most 16 reference images")
	})

	t.Run("multipart uploads", func(t *testing.T) {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		require.NoError(t, writer.WriteField("model", "gpt-image-2"))
		require.NoError(t, writer.WriteField("prompt", "cat"))
		for i := 0; i < 17; i++ {
			part, err := writer.CreateFormFile("image[]", fmt.Sprintf("%d.png", i))
			require.NoError(t, err)
			_, err = part.Write([]byte("image"))
			require.NoError(t, err)
		}
		require.NoError(t, writer.Close())

		_, err := parseOpenAIImagesForValidation(t, "/v1/images/edits", writer.FormDataContentType(), body.Bytes())
		require.ErrorContains(t, err, "images must contain at most 16 reference images")
	})
}

func parseOpenAIImagesForValidation(t *testing.T, path, contentType string, body []byte) (*OpenAIImagesRequest, error) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", contentType)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = req
	return (&OpenAIGatewayService{}).ParseOpenAIImagesRequest(ctx, body)
}
