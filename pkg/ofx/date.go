package ofx

import (
	"fmt"
	"strings"
	"time"
)

// ParseDate parses an OFX date string into a time.Time value.
// Supported formats:
//   - YYYYMMDD
//   - YYYYMMDDHHMMSS
//   - YYYYMMDDHHMMSS.XXX
//   - Any of the above with timezone suffix [-N:TZ] (e.g., "[-3:BRT]")
func ParseDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("%w: empty string", ErrInvalidDate)
	}

	// Strip timezone bracket suffix if present: "20260115120000[-3:BRT]"
	loc := time.UTC
	if idx := strings.Index(s, "["); idx != -1 {
		tzStr := s[idx:]
		s = s[:idx]

		parsedLoc, tzErr := parseOFXTimezone(tzStr)
		if tzErr == nil {
			loc = parsedLoc
		}
	}

	// Remove fractional seconds if present: "20260115120000.000" → "20260115120000"
	if idx := strings.Index(s, "."); idx != -1 {
		s = s[:idx]
	}

	var t time.Time
	var parseErr error

	switch len(s) {
	case 8:
		// YYYYMMDD
		t, parseErr = time.ParseInLocation("20060102", s, loc)
	case 14:
		// YYYYMMDDHHMMSS
		t, parseErr = time.ParseInLocation("20060102150405", s, loc)
	default:
		return time.Time{}, fmt.Errorf("%w: unexpected length %d for %q", ErrInvalidDate, len(s), s)
	}

	if parseErr != nil {
		return time.Time{}, fmt.Errorf("%w: %s", ErrInvalidDate, parseErr.Error())
	}

	return t.UTC(), nil
}

// parseOFXTimezone parses a timezone bracket like "[-3:BRT]" or "[5:EST]".
func parseOFXTimezone(s string) (*time.Location, error) {
	s = strings.TrimPrefix(s, "[")
	s = strings.TrimSuffix(s, "]")

	// Split on ":" — first part is offset hours, second is name
	parts := strings.SplitN(s, ":", 2)
	if len(parts) == 0 || parts[0] == "" {
		return time.UTC, nil
	}

	var offsetHours int
	_, scanErr := fmt.Sscanf(parts[0], "%d", &offsetHours)
	if scanErr != nil {
		return time.UTC, fmt.Errorf("parsing timezone offset: %w", scanErr)
	}

	name := "UTC"
	if len(parts) == 2 && parts[1] != "" {
		name = parts[1]
	}

	return time.FixedZone(name, offsetHours*3600), nil
}
