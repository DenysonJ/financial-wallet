package httputil

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func FuzzWriteErrorWithDetails(f *testing.F) {
	f.Add(400, "VALIDATION", "bad request", "field", "email")                            // normal
	f.Add(100, "", "", "", "")                                                           // minimal valid status
	f.Add(999, "CODE", "message", "key", "value")                                        // unusual status
	f.Add(200, "OK", "not an error", "detail", "val")                                    // 200 with error format
	f.Add(500, string(make([]byte, 10000)), string(make([]byte, 10000)), "\x00", "\x00") // long + null
	f.Add(200, "コード", "メッセージ", "キー", "値")                                                // unicode

	f.Fuzz(func(t *testing.T, status int, code, message, detailKey, detailValue string) {
		// httptest.ResponseRecorder panics on invalid status codes (< 100 or > 999)
		if status < 100 || status > 999 {
			t.Skip("invalid HTTP status code")
		}

		w := httptest.NewRecorder()

		details := map[string]any{detailKey: detailValue}
		WriteErrorWithDetails(w, status, code, message, details)

		// Must never panic
		body := w.Body.Bytes()

		// Response body must be valid JSON
		assert.True(t, json.Valid(body), "response must be valid JSON")

		// Content-Type must be set
		assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

		// Status code must match
		assert.Equal(t, status, w.Code)

		// Must deserialize into expected structure
		var resp ErrorResponse
		unmarshalErr := json.Unmarshal(body, &resp)
		assert.NoError(t, unmarshalErr)
		assert.Equal(t, message, resp.Errors.Message)
		assert.Equal(t, code, resp.Errors.Code)
	})
}
