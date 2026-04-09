package logutil

import (
	"log/slog"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
)

func FuzzMaskEmail(f *testing.F) {
	f.Add("user@example.com")          // valid
	f.Add("a@b.c")                     // minimal valid
	f.Add("")                          // empty
	f.Add("no-at-sign")                // no @
	f.Add("@leading")                  // @ at start
	f.Add("trailing@")                 // @ at end
	f.Add("multi@@ats")                // double @
	f.Add("user@例え.jp")                // unicode domain
	f.Add("名前@example.com")            // multibyte local part
	f.Add(string(make([]byte, 10000))) // very long
	f.Add("user@example.com\x00")      // null byte
	f.Add("a@b")                       // minimal with @
	f.Add("   @   ")                   // whitespace around @

	f.Fuzz(func(t *testing.T, input string) {
		result := MaskEmail(input)

		// Must never panic — if we got here, it didn't
		assert.NotEmpty(t, result, "result must never be empty")

		atIdx := strings.LastIndex(input, "@")
		if input == "" || atIdx <= 0 {
			// No valid @ position → fully masked
			assert.Equal(t, "***", result)
		} else {
			// Has @ at valid position → result must contain @
			assert.Contains(t, result, "@")
			assert.Contains(t, result, "***")
		}
	})
}

func FuzzMaskDocument(f *testing.F) {
	f.Add("12345678901")               // CPF
	f.Add("1234")                      // exactly 4 bytes
	f.Add("12345")                     // exactly 5 bytes
	f.Add("123")                       // under 4
	f.Add("")                          // empty
	f.Add("a")                         // single char
	f.Add(string(make([]byte, 10000))) // very long
	f.Add("doc\x00null")               // null byte
	f.Add("日本語文字")                     // multibyte CJK (15 bytes, 5 runes)

	f.Fuzz(func(t *testing.T, input string) {
		result := MaskDocument(input)

		// Must never panic
		assert.NotEmpty(t, result)

		if len(input) <= 4 {
			assert.Equal(t, "***", result)
		} else {
			// Must start with *** and end with last 4 bytes
			assert.True(t, strings.HasPrefix(result, "***"))
			expectedSuffix := input[len(input)-4:]
			assert.Equal(t, "***"+expectedSuffix, result)
		}
	})
}

func FuzzMaskName(f *testing.F) {
	f.Add("Joao Silva")                // normal name
	f.Add("")                          // empty
	f.Add("  ")                        // whitespace only
	f.Add("A")                         // single char
	f.Add("名前 苗字")                     // CJK
	f.Add("a b c")                     // single-char words
	f.Add(string(make([]byte, 10000))) // very long
	f.Add("Name\x00With\x00Nulls")     // null bytes
	f.Add("   Lots   Of   Spaces   ")  // extra whitespace
	f.Add("\t\n")                      // tab/newline
	f.Add("Ã")                         // single multibyte rune

	f.Fuzz(func(t *testing.T, input string) {
		result := MaskName(input)

		// Must never panic
		assert.NotEmpty(t, result)

		trimmed := strings.TrimSpace(input)
		if trimmed == "" {
			assert.Equal(t, "***", result)
			return
		}

		// Number of parts in result must equal number of fields in input
		inputParts := strings.Fields(trimmed)
		resultParts := strings.Fields(result)
		assert.Equal(t, len(inputParts), len(resultParts))

		// Each result part must start with the first rune of the input part
		for i, part := range inputParts {
			if i >= len(resultParts) {
				break
			}
			r, _ := utf8.DecodeRuneInString(part)
			if r == utf8.RuneError || utf8.RuneCountInString(part) <= 1 {
				// Single-rune or invalid → kept as-is
				assert.Equal(t, part, resultParts[i])
			} else {
				// Multi-rune → first rune + "***"
				assert.Equal(t, string(r)+"***", resultParts[i])
			}
		}
	})
}

func FuzzMaskPhone(f *testing.F) {
	f.Add("+5511999998888")            // BR mobile
	f.Add("+14155551234")              // US
	f.Add("")                          // empty
	f.Add("123456")                    // 6 digits (under 7)
	f.Add("1234567")                   // exactly 7 digits
	f.Add("+0000000")                  // 7 zeros
	f.Add("no-digits-here")            // no digits
	f.Add(string(make([]byte, 10000))) // very long
	f.Add("+55\x00119999")             // null byte
	f.Add("(011) 9999-8888")           // formatted with parens/hyphens
	f.Add("+")                         // plus only
	f.Add("++1234567890")              // double plus

	f.Fuzz(func(t *testing.T, input string) {
		result := MaskPhone(input)

		// Must never panic
		assert.NotEmpty(t, result)

		// Count digits in input
		digits := make([]byte, 0, len(input))
		hasPlus := false
		for i, ch := range input {
			if ch == '+' && i == 0 {
				hasPlus = true
				continue
			}
			if ch >= '0' && ch <= '9' {
				digits = append(digits, byte(ch))
			}
		}

		if input == "" || len(digits) < 7 {
			assert.Equal(t, "***", result)
		} else {
			assert.Contains(t, result, "***")

			// Must contain the first 2 digits and last 4 digits
			first2 := string(digits[:2])
			last4 := string(digits[len(digits)-4:])
			assert.Contains(t, result, first2)
			assert.True(t, strings.HasSuffix(result, last4))

			// Check + prefix consistency
			if hasPlus {
				assert.True(t, strings.HasPrefix(result, "+"))
			} else {
				assert.False(t, strings.HasPrefix(result, "+"))
			}
		}
	})
}

func FuzzMaskAttr(f *testing.F) {
	f.Add("email", "user@example.com")                            // sensitive field
	f.Add("name", "Joao Silva")                                   // sensitive field
	f.Add("document", "12345678901")                              // sensitive field
	f.Add("phone", "+5511999998888")                              // sensitive field
	f.Add("unknown_field", "some value")                          // non-sensitive
	f.Add("", "")                                                 // empty key and value
	f.Add("EMAIL", "User@Example.com")                            // uppercase key
	f.Add("email", "")                                            // sensitive but empty value
	f.Add(string(make([]byte, 1000)), string(make([]byte, 1000))) // very long
	f.Add("email\x00", "user@example.com")                        // null in key
	f.Add("名前", "日本語")                                            // unicode key/value

	masker := NewMasker(DefaultBRConfig())
	handler := NewMaskingHandler(masker, slog.Default().Handler())

	f.Fuzz(func(t *testing.T, key, value string) {
		attr := slog.String(key, value)
		result := handler.maskAttr(attr)

		// Must never panic
		assert.Equal(t, key, result.Key, "key must be preserved")

		normalizedKey := strings.ToLower(key)
		_, isSensitive := masker.config.Fields[normalizedKey]

		if isSensitive && value != "" {
			resultStr := result.Value.String()
			// Sensitive non-empty values must be masked: either the value
			// changes or the fallback "***" is applied (which may equal input).
			assert.True(t, resultStr != value || resultStr == "***",
				"sensitive field %q must be masked", key)
		} else if !isSensitive {
			// Non-sensitive values must pass through unchanged
			assert.Equal(t, value, result.Value.String(),
				"non-sensitive field %q must not be modified", key)
		}
	})
}
