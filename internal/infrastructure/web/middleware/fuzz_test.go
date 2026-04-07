package middleware

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func FuzzParseServiceKeys(f *testing.F) {
	f.Add("service1:key1,service2:key2") // multiple pairs
	f.Add("service:key")                 // single pair
	f.Add("")                            // empty
	f.Add("no-colon")                    // no colon
	f.Add(":::")                         // only colons
	f.Add(",,,,")                        // only commas
	f.Add("a:b,,c:d")                    // empty segment between commas
	f.Add(",service:key,")               // leading/trailing commas
	f.Add("  name  :  key  ")            // whitespace
	f.Add("svc:key1:key2")               // multiple colons (SplitN 2)
	f.Add(":key")                        // empty name
	f.Add("name:")                       // empty key
	f.Add(string(make([]byte, 10000)))   // very long
	f.Add("svc\x00:key\x00")             // null bytes
	f.Add("名前:キー")                       // unicode
	f.Add("a:b,a:c")                     // duplicate names

	f.Fuzz(func(t *testing.T, input string) {
		result := ParseServiceKeys(input)

		// Must never panic
		assert.NotNil(t, result, "result map must never be nil")

		for k, v := range result {
			// Keys and values must never be empty
			assert.NotEmpty(t, k, "service name must not be empty")
			assert.NotEmpty(t, v, "service key must not be empty")

			// Keys and values must have no leading/trailing whitespace
			assert.Equal(t, strings.TrimSpace(k), k, "service name must be trimmed")
			assert.Equal(t, strings.TrimSpace(v), v, "service key must be trimmed")
		}
	})
}

func FuzzBodyFingerprint(f *testing.F) {
	f.Add([]byte("{}"))              // empty JSON
	f.Add([]byte(""))                // empty
	f.Add([]byte("\x00\x00\x00"))    // null bytes
	f.Add(make([]byte, 10000))       // large body
	f.Add([]byte(`{"key":"value"}`)) // valid JSON
	f.Add([]byte("not json at all")) // arbitrary text

	hexPattern := regexp.MustCompile(`^[0-9a-f]{64}$`)

	f.Fuzz(func(t *testing.T, body []byte) {
		result := bodyFingerprint(body)

		// Must never panic
		// SHA-256 always produces 64 hex characters
		assert.Len(t, result, 64, "SHA-256 hex must be 64 chars")
		assert.True(t, hexPattern.MatchString(result), "must be valid hex: %s", result)

		// Deterministic: same input → same output
		result2 := bodyFingerprint(body)
		assert.Equal(t, result, result2, "fingerprint must be deterministic")
	})
}

func FuzzBuildIdempotencyKey(f *testing.F) {
	f.Add("service1", "key1")                                       // normal
	f.Add("", "key1")                                               // empty service
	f.Add("", "")                                                   // both empty
	f.Add("svc", "")                                                // empty key
	f.Add(string(make([]byte, 10000)), string(make([]byte, 10000))) // very long
	f.Add("svc\x00", "key\x00")                                     // null bytes
	f.Add("サービス", "キー")                                             // unicode
	f.Add("svc:with:colons", "key:with:colons")                     // colons in input

	f.Fuzz(func(t *testing.T, serviceName, key string) {
		result := buildIdempotencyKey(serviceName, key)

		// Must never panic
		assert.True(t, strings.HasPrefix(result, "idempotency:"),
			"must start with 'idempotency:' prefix")

		if serviceName != "" {
			expected := "idempotency:" + serviceName + ":" + key
			assert.Equal(t, expected, result)
		} else {
			expected := "idempotency:" + key
			assert.Equal(t, expected, result)
		}
	})
}

func FuzzValidRequestID(f *testing.F) {
	f.Add("abc-123")                        // valid
	f.Add("")                               // empty
	f.Add("a")                              // single char
	f.Add(string(make([]byte, 64)))         // exactly max len (null bytes)
	f.Add(string(make([]byte, 65)))         // over max len
	f.Add("has spaces")                     // spaces
	f.Add("special!@#$")                    // special chars
	f.Add("550e8400-e29b-41d4-a716-446655") // UUID-like
	f.Add("\x00")                           // null byte
	f.Add("名前")                             // unicode
	f.Add("ABCDEF-0123456789")              // uppercase + digits + hyphen
	f.Add("---")                            // only hyphens

	// Regex to manually verify: only [a-zA-Z0-9-]
	manualPattern := regexp.MustCompile(`^[a-zA-Z0-9-]+$`)

	f.Fuzz(func(t *testing.T, input string) {
		regexMatch := validRequestID.MatchString(input)
		lenOk := len(input) <= requestIDMaxLen

		// Must never panic

		// Regex result must be consistent with manual check
		expectedMatch := manualPattern.MatchString(input)
		assert.Equal(t, expectedMatch, regexMatch,
			"regex match must be consistent for input %q", input)

		// Combined validation: accepted only if length OK AND regex matches
		accepted := lenOk && regexMatch
		if accepted {
			assert.LessOrEqual(t, len(input), 64)
			assert.True(t, regexMatch)
		}
	})
}
