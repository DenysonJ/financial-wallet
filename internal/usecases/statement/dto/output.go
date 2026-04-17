package dto

// =============================================================================
// Statement Output DTOs
// =============================================================================

// StatementOutput represents a single statement in API responses.
type StatementOutput struct {
	ID           string  `json:"id"`
	AccountID    string  `json:"account_id"`
	Type         string  `json:"type"`
	Amount       int64   `json:"amount"`
	Description  string  `json:"description"`
	ReferenceID  *string `json:"reference_id,omitempty"`
	ExternalID   *string `json:"external_id,omitempty"`
	BalanceAfter int64   `json:"balance_after"`
	PostedAt     string  `json:"posted_at"`
	CreatedAt    string  `json:"created_at"`
}

// CreateOutput represents the output after creating a statement.
type CreateOutput = StatementOutput

// GetOutput represents the output when fetching a single statement.
type GetOutput = StatementOutput

// ListOutput represents the paginated output for listing statements.
type ListOutput struct {
	Data       []StatementOutput `json:"data"`
	Pagination PaginationOutput  `json:"pagination"`
}

// PaginationOutput represents pagination metadata.
type PaginationOutput struct {
	Page       int     `json:"page"`
	Limit      int     `json:"limit"`
	Total      int     `json:"total"`
	TotalPages int     `json:"total_pages"`
	NextCursor *string `json:"next_cursor,omitempty"`
}

// ImportOutput represents the result of an OFX file import.
type ImportOutput struct {
	TotalTransactions int `json:"total_transactions"`
	Created           int `json:"created"`
	Skipped           int `json:"skipped"`
}
