package config

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func FuzzGetEnvBool(f *testing.F) {
	f.Add("true")                      // canonical true
	f.Add("false")                     // canonical false
	f.Add("TRUE")                      // uppercase
	f.Add("1")                         // numeric true
	f.Add("0")                         // numeric false
	f.Add("")                          // empty (env var set but empty)
	f.Add("not-a-bool")                // invalid
	f.Add("yes")                       // common but invalid for ParseBool
	f.Add("no")                        // common but invalid for ParseBool
	f.Add(string(make([]byte, 10000))) // very long
	f.Add("true\x00")                  // null byte
	f.Add("  true  ")                  // whitespace (ParseBool doesn't trim)
	f.Add("T")                         // single char valid for ParseBool

	f.Fuzz(func(t *testing.T, input string) {
		// Skip inputs with null bytes — os.Setenv rejects them
		if strings.ContainsRune(input, 0) {
			t.Skip("null bytes not allowed in env values")
		}

		const envKey = "FUZZ_BOOL_KEY"
		t.Setenv(envKey, input)

		resultFalse := getEnvBool(envKey, false)
		resultTrue := getEnvBool(envKey, true)

		// Must never panic

		// If input is empty, getEnvBool skips parsing and returns fallback
		if input == "" {
			assert.Equal(t, false, resultFalse, "empty input must return fallback=false")
			assert.Equal(t, true, resultTrue, "empty input must return fallback=true")
			return
		}

		// If ParseBool succeeds, both calls must return the same parsed value
		parsed, parseErr := strconv.ParseBool(input)
		if parseErr == nil {
			assert.Equal(t, parsed, resultFalse)
			assert.Equal(t, parsed, resultTrue)
		} else {
			// Parse failed → must return respective fallbacks
			assert.Equal(t, false, resultFalse, "parse error must return fallback=false")
			assert.Equal(t, true, resultTrue, "parse error must return fallback=true")
		}
	})
}

func FuzzGetEnvInt(f *testing.F) {
	f.Add("42")                        // normal
	f.Add("0")                         // zero
	f.Add("-1")                        // negative
	f.Add("")                          // empty
	f.Add("not-a-number")              // invalid
	f.Add("9999999999999999999")       // overflow
	f.Add("1.5")                       // float
	f.Add(string(make([]byte, 10000))) // very long
	f.Add("42\x00")                    // null byte
	f.Add("  42  ")                    // whitespace (Atoi doesn't trim)
	f.Add("0x1A")                      // hex

	f.Fuzz(func(t *testing.T, input string) {
		if strings.ContainsRune(input, 0) {
			t.Skip("null bytes not allowed in env values")
		}

		const envKey = "FUZZ_INT_KEY"
		t.Setenv(envKey, input)

		fallback := 99
		result := getEnvInt(envKey, fallback)

		// Must never panic

		if input == "" {
			assert.Equal(t, fallback, result, "empty input must return fallback")
			return
		}

		parsed, parseErr := strconv.Atoi(input)
		if parseErr == nil {
			assert.Equal(t, parsed, result)
		} else {
			assert.Equal(t, fallback, result, "parse error must return fallback")
		}
	})
}

func FuzzGetEnvDuration(f *testing.F) {
	f.Add("5m")                        // minutes
	f.Add("1h")                        // hours
	f.Add("30s")                       // seconds
	f.Add("100ms")                     // milliseconds
	f.Add("")                          // empty
	f.Add("not-a-duration")            // invalid
	f.Add("-5m")                       // negative
	f.Add("0")                         // zero
	f.Add(string(make([]byte, 10000))) // very long
	f.Add("5m\x00")                    // null byte
	f.Add("1h30m5s")                   // compound
	f.Add("999999999h")                // very large

	f.Fuzz(func(t *testing.T, input string) {
		if strings.ContainsRune(input, 0) {
			t.Skip("null bytes not allowed in env values")
		}

		const envKey = "FUZZ_DURATION_KEY"
		t.Setenv(envKey, input)

		fallback := 5 * time.Minute
		result := getEnvDuration(envKey, fallback)

		// Must never panic

		if input == "" {
			assert.Equal(t, fallback, result, "empty input must return fallback")
			return
		}

		parsed, parseErr := time.ParseDuration(input)
		if parseErr == nil {
			assert.Equal(t, parsed, result)
		} else {
			assert.Equal(t, fallback, result, "parse error must return fallback")
		}
	})
}
