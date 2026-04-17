package ofx

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// ParseAmount converts a signed OFX decimal string to cents (int64).
// Examples: "150.75" → 15075, "-30.50" → -3050, "100" → 10000, "-7.5" → -750.
func ParseAmount(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("%w: empty string", ErrInvalidAmount)
	}

	// Determine sign and strip it for processing
	negative := false
	if s[0] == '-' {
		negative = true
		s = s[1:]
	} else if s[0] == '+' {
		s = s[1:]
	}

	if s == "" {
		return 0, fmt.Errorf("%w: only sign character", ErrInvalidAmount)
	}

	var cents int64

	parts := strings.SplitN(s, ".", 2)
	switch len(parts) {
	case 1:
		// No decimal point: "100" → 10000
		whole, wholeErr := strconv.ParseInt(parts[0], 10, 64)
		if wholeErr != nil {
			return 0, fmt.Errorf("%w: %s", ErrInvalidAmount, wholeErr.Error())
		}
		cents = whole * 100
	case 2:
		whole, wholeErr := strconv.ParseInt(parts[0], 10, 64)
		if wholeErr != nil {
			return 0, fmt.Errorf("%w: %s", ErrInvalidAmount, wholeErr.Error())
		}

		fracStr := parts[1]
		if len(fracStr) == 0 {
			// "100." → 10000
			cents = whole * 100
		} else if len(fracStr) <= 2 {
			// Pad to 2 digits: "7.5" → "50", "7.05" → "05"
			for len(fracStr) < 2 {
				fracStr += "0"
			}
			frac, fracErr := strconv.ParseInt(fracStr, 10, 64)
			if fracErr != nil {
				return 0, fmt.Errorf("%w: %s", ErrInvalidAmount, fracErr.Error())
			}
			cents = whole*100 + frac
		} else {
			// More than 2 decimal places: parse as float and round
			f, fErr := strconv.ParseFloat(parts[0]+"."+fracStr, 64)
			if fErr != nil {
				return 0, fmt.Errorf("%w: %s", ErrInvalidAmount, fErr.Error())
			}
			cents = int64(math.Round(f * 100))
		}
	}

	if negative {
		cents = -cents
	}

	return cents, nil
}
