package dto

import "io"

// =============================================================================
// Statement Input DTOs
// =============================================================================

// CreateInput represents the input data for creating a statement (credit or debit).
type CreateInput struct {
	AccountID        string   `json:"-"`                                                  // From URL path
	RequestingUserID string   `json:"-"`                                                  // From JWT context
	Type             string   `json:"type" binding:"required,oneof=credit debit"`         // credit or debit
	Amount           int64    `json:"amount" binding:"required,gt=0"`                     // Amount in cents (positive)
	Description      string   `json:"description,omitempty" binding:"omitempty,max=1000"` // Optional description
	PostedAt         string   `json:"posted_at" binding:"omitempty"`
	CategoryID       *string  `json:"category_id,omitempty"` // Optional UUID; type must match Type
	TagIDs           []string `json:"tag_ids,omitempty"`     // Optional UUIDs; deduped, max 10
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
	AccountID        string   `form:"-"`           // Account UUID from URL path
	RequestingUserID string   `form:"-"`           // From JWT context
	Type             string   `form:"type"`        // Optional filter: credit or debit
	DateFrom         string   `form:"date_from"`   // Optional filter: RFC3339 date
	DateTo           string   `form:"date_to"`     // Optional filter: RFC3339 date
	CategoryID       string   `form:"category_id"` // Optional filter: only statements with this category
	TagIDs           []string `form:"tag_ids"`     // Optional filter: statements with ANY of these tag IDs (repeated query param)
	Page             int      `form:"page"`        // Page number (1-indexed, used when cursor is empty)
	Limit            int      `form:"limit"`       // Items per page
	Cursor           string   `form:"cursor"`      // Opaque cursor for keyset pagination (overrides page)
}

// ImportOFXInput represents the input data for importing an OFX file.
type ImportOFXInput struct {
	AccountID        string    // Account UUID from URL path
	RequestingUserID string    // From JWT context
	FileContent      io.Reader // OFX file content
}

// UpdateCategoryInput — body for PATCH /statements/:id/category.
//
// CategoryID is a pointer so we can distinguish "set X" from "clear":
//   - {"category_id": "<uuid>"} → set/swap
//   - {"category_id": null}     → clear
type UpdateCategoryInput struct {
	StatementID      string  `json:"-"` // From URL path
	RequestingUserID string  `json:"-"` // From JWT context
	CategoryID       *string `json:"category_id"`
}

// ReplaceTagsInput — body for PUT /statements/:id/tags.
// Replaces the entire tag set with the array sent (empty slice clears all).
type ReplaceTagsInput struct {
	StatementID      string   `json:"-"` // From URL path
	RequestingUserID string   `json:"-"` // From JWT context
	TagIDs           []string `json:"tag_ids"`
}
