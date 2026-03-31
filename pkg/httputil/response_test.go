package httputil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteSuccess(t *testing.T) {
	t.Run("returns data with status 200", func(t *testing.T) {
		w := httptest.NewRecorder()
		WriteSuccess(w, http.StatusOK, map[string]any{"id": "123", "name": "Test"})

		assert.Equal(t, http.StatusOK, w.Code)

		var resp SuccessResponse
		parseErr := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, parseErr)
		assert.NotNil(t, resp.Data)
	})

	t.Run("returns nil data", func(t *testing.T) {
		w := httptest.NewRecorder()
		WriteSuccess(w, http.StatusOK, nil)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]any
		parseErr := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, parseErr)
		assert.Nil(t, resp["data"])
	})

	t.Run("returns status 201 Created", func(t *testing.T) {
		w := httptest.NewRecorder()
		WriteSuccess(w, http.StatusCreated, map[string]any{"id": "456"})

		assert.Equal(t, http.StatusCreated, w.Code)

		var resp SuccessResponse
		parseErr := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, parseErr)
		assert.NotNil(t, resp.Data)
	})

	t.Run("returns status 204 No Content with nil data", func(t *testing.T) {
		w := httptest.NewRecorder()
		WriteSuccess(w, http.StatusNoContent, nil)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("response Content-Type is application/json", func(t *testing.T) {
		w := httptest.NewRecorder()
		WriteSuccess(w, http.StatusOK, map[string]any{"ok": true})

		assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
	})

	t.Run("meta and links are omitted when not provided", func(t *testing.T) {
		w := httptest.NewRecorder()
		WriteSuccess(w, http.StatusOK, map[string]any{"id": "1"})

		var raw map[string]any
		parseErr := json.Unmarshal(w.Body.Bytes(), &raw)
		require.NoError(t, parseErr)
		_, hasMeta := raw["meta"]
		_, hasLinks := raw["links"]
		assert.False(t, hasMeta, "meta should be omitted")
		assert.False(t, hasLinks, "links should be omitted")
	})
}

func TestWriteError(t *testing.T) {
	statusCases := []struct {
		name    string
		status  int
		message string
	}{
		{"400 Bad Request", http.StatusBadRequest, "invalid request"},
		{"401 Unauthorized", http.StatusUnauthorized, "authentication required"},
		{"403 Forbidden", http.StatusForbidden, "access denied"},
		{"404 Not Found", http.StatusNotFound, "resource not found"},
		{"500 Internal Server Error", http.StatusInternalServerError, "internal error"},
	}

	for _, tc := range statusCases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			WriteError(w, tc.status, tc.message)

			assert.Equal(t, tc.status, w.Code)
			assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

			var resp ErrorResponse
			parseErr := json.Unmarshal(w.Body.Bytes(), &resp)
			require.NoError(t, parseErr)
			assert.Equal(t, tc.message, resp.Errors.Message)
		})
	}
}

func TestWriteSuccessWithMeta(t *testing.T) {
	t.Run("with meta and links populated", func(t *testing.T) {
		w := httptest.NewRecorder()
		data := []string{"a", "b"}
		meta := map[string]any{"total": 2, "page": 1}
		links := map[string]any{"next": "/test?page=2", "prev": "/test?page=0"}
		WriteSuccessWithMeta(w, http.StatusOK, data, meta, links)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]any
		parseErr := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, parseErr)
		assert.NotNil(t, resp["data"])
		assert.NotNil(t, resp["meta"])
		assert.NotNil(t, resp["links"])

		metaMap, ok := resp["meta"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, float64(2), metaMap["total"])
		assert.Equal(t, float64(1), metaMap["page"])

		linksMap, ok := resp["links"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "/test?page=2", linksMap["next"])
	})

	t.Run("with nil meta", func(t *testing.T) {
		w := httptest.NewRecorder()
		WriteSuccessWithMeta(w, http.StatusOK, []string{"x"}, nil, map[string]any{"self": "/test"})

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]any
		parseErr := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, parseErr)
		_, hasMeta := resp["meta"]
		assert.False(t, hasMeta, "nil meta should be omitted due to omitempty")
		assert.NotNil(t, resp["links"])
	})

	t.Run("with nil links", func(t *testing.T) {
		w := httptest.NewRecorder()
		WriteSuccessWithMeta(w, http.StatusOK, []string{"x"}, map[string]any{"total": 1}, nil)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]any
		parseErr := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, parseErr)
		assert.NotNil(t, resp["meta"])
		_, hasLinks := resp["links"]
		assert.False(t, hasLinks, "nil links should be omitted due to omitempty")
	})

	t.Run("with both meta and links nil", func(t *testing.T) {
		w := httptest.NewRecorder()
		WriteSuccessWithMeta(w, http.StatusOK, map[string]any{"id": "1"}, nil, nil)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]any
		parseErr := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, parseErr)
		assert.NotNil(t, resp["data"])
		_, hasMeta := resp["meta"]
		_, hasLinks := resp["links"]
		assert.False(t, hasMeta, "nil meta should be omitted")
		assert.False(t, hasLinks, "nil links should be omitted")
	})

	t.Run("response Content-Type is application/json", func(t *testing.T) {
		w := httptest.NewRecorder()
		WriteSuccessWithMeta(w, http.StatusOK, "data", map[string]any{}, map[string]any{})

		assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
	})
}

func TestWriteErrorWithCode(t *testing.T) {
	t.Run("includes error code in response", func(t *testing.T) {
		w := httptest.NewRecorder()
		WriteErrorWithCode(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "field is required")

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)

		var resp ErrorResponse
		parseErr := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, parseErr)
		assert.Equal(t, "VALIDATION_ERROR", resp.Errors.Code)
		assert.Equal(t, "field is required", resp.Errors.Message)
	})

	t.Run("response Content-Type is application/json", func(t *testing.T) {
		w := httptest.NewRecorder()
		WriteErrorWithCode(w, http.StatusBadRequest, "BAD_REQUEST", "bad")

		assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
	})
}

func TestWriteErrorWithDetails(t *testing.T) {
	t.Run("includes code and details in response", func(t *testing.T) {
		details := map[string]any{
			"field":  "email",
			"reason": "invalid format",
		}
		w := httptest.NewRecorder()
		WriteErrorWithDetails(w, http.StatusBadRequest, "VALIDATION_ERROR", "validation failed", details)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp ErrorResponse
		parseErr := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, parseErr)
		assert.Equal(t, "VALIDATION_ERROR", resp.Errors.Code)
		assert.Equal(t, "validation failed", resp.Errors.Message)
		assert.Equal(t, "email", resp.Errors.Details["field"])
		assert.Equal(t, "invalid format", resp.Errors.Details["reason"])
	})

	t.Run("with nil details", func(t *testing.T) {
		w := httptest.NewRecorder()
		WriteErrorWithDetails(w, http.StatusBadRequest, "ERR", "error msg", nil)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var raw map[string]any
		parseErr := json.Unmarshal(w.Body.Bytes(), &raw)
		require.NoError(t, parseErr)

		errorsMap, ok := raw["errors"].(map[string]any)
		require.True(t, ok)
		_, hasDetails := errorsMap["details"]
		assert.False(t, hasDetails, "nil details should be omitted due to omitempty")
	})
}
