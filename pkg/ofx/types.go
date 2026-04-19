package ofx

import "time"

// Transaction represents a single financial transaction extracted from an OFX file.
type Transaction struct {
	FITID      string
	Type       string
	Amount     int64 // Signed amount in cents (positive = credit, negative = debit)
	Name       string
	Memo       string
	DatePosted time.Time
}

// Header contains OFX file metadata parsed from the header block.
type Header struct {
	Version  string // e.g., "102", "200"
	Encoding string // e.g., "USASCII", "UTF-8"
}

// ParseResult holds the complete output of parsing an OFX file.
type ParseResult struct {
	Header       Header
	Transactions []Transaction
}
