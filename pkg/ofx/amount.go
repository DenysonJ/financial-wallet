package ofx

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// maxWholePart is the largest whole number that can be multiplied by 100 without int64 overflow.
const maxWholePart = math.MaxInt64 / 100

// ParseAmount converts a signed OFX decimal string to cents (int64).
// Examples: "150.75" → 15075, "-30.50" → -3050, "100" → 10000, "-7.5" → -750.
func ParseAmount(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("%w: empty string", ErrInvalidAmount)
	}

	// Determine sign and strip it for processing
	negative := false
	switch s[0] {
	case '-':
		negative = true
		s = s[1:]
	case '+':
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
		if whole > maxWholePart {
			return 0, fmt.Errorf("%w: amount too large", ErrInvalidAmount)
		}
		cents = whole * 100
	case 2:
		whole, wholeErr := strconv.ParseInt(parts[0], 10, 64)
		if wholeErr != nil {
			return 0, fmt.Errorf("%w: %s", ErrInvalidAmount, wholeErr.Error())
		}
		if whole > maxWholePart {
			return 0, fmt.Errorf("%w: amount too large", ErrInvalidAmount)
		}

		fracStr := parts[1]
		switch {
		case fracStr == "":
			// "100." → 10000
			cents = whole * 100
		case len(fracStr) <= 2:
			// Pad to 2 digits: "7.5" → "50", "7.05" → "05"
			for len(fracStr) < 2 {
				fracStr += "0"
			}
			frac, fracErr := strconv.ParseInt(fracStr, 10, 64)
			if fracErr != nil {
				return 0, fmt.Errorf("%w: %s", ErrInvalidAmount, fracErr.Error())
			}
			cents = whole*100 + frac
		default:
			// More than 2 decimal places: parse as float and round
			f, fErr := strconv.ParseFloat(parts[0]+"."+fracStr, 64)
			if fErr != nil {
				return 0, fmt.Errorf("%w: %s", ErrInvalidAmount, fErr.Error())
			}
			rounded := math.Round(f * 100)
			if rounded > float64(math.MaxInt64) || rounded < float64(math.MinInt64) {
				return 0, fmt.Errorf("%w: amount too large", ErrInvalidAmount)
			}
			cents = int64(rounded)
		}
	}

	if negative {
		cents = -cents
	}

	return cents, nil
}
