package dto

import "io"

// =============================================================================
// Statement Input DTOs
// =============================================================================

// CreateInput represents the input data for creating a statement (credit or debit).
type CreateInput struct {
	AccountID        string `json:"-"`                                                  // From URL path
	RequestingUserID string `json:"-"`                                                  // From JWT context
	Type             string `json:"type" binding:"required,oneof=credit debit"`         // credit or debit
	Amount           int64  `json:"amount" binding:"required,gt=0"`                     // Amount in cents (positive)
	Description      string `json:"description,omitempty" binding:"omitempty,max=1000"` // Optional description
	PostedAt         string `json:"posted_at" binding:"omitempty"`
}

// ReverseInput represents the input data for reversing a statement.
type ReverseInput struct {
	StatementID      string `json:"-"`                                                  // From URL path
	AccountID        string `json:"-"`                                                  // From URL path
	RequestingUserID string `json:"-"`                                                  // From JWT context
	Description      string `json:"description,omitempty" binding:"omitempty,max=1000"` // Optional reversal description
}

// GetInput represents the input data for fetching a single statement.
type GetInput struct {
	ID               string `json:"-"` // Statement UUID from URL path
	AccountID        string `json:"-"` // Account UUID from URL path
	RequestingUserID string `json:"-"` // From JWT context
}

// ListInput represents the input data for listing statements by account.
type ListInput struct {
	AccountID        string `form:"-"`         // Account UUID from URL path
	RequestingUserID string `form:"-"`         // From JWT context
	Type             string `form:"type"`      // Optional filter: credit or debit
	DateFrom         string `form:"date_from"` // Optional filter: RFC3339 date
	DateTo           string `form:"date_to"`   // Optional filter: RFC3339 date
	Page             int    `form:"page"`      // Page number (1-indexed)
	Limit            int    `form:"limit"`     // Items per page
}

// ImportOFXInput represents the input data for importing an OFX file.
type ImportOFXInput struct {
	AccountID        string    // Account UUID from URL path
	RequestingUserID string    // From JWT context
	FileContent      io.Reader // OFX file content
}
