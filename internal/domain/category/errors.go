package category

import "errors"

// Domain errors for Category.
var (
	ErrCategoryNotFound     = errors.New("category not found")
	ErrCategoryDuplicate    = errors.New("category already exists")
	ErrCategoryReadOnly     = errors.New("system category is read-only")
	ErrCategoryInUse        = errors.New("category is in use by one or more statements")
	ErrCategoryTypeMismatch = errors.New("category type does not match statement type")
	// ErrCategoryNotVisible Returned instead of NotFound to avoid cross-user existence oracle.
	ErrCategoryNotVisible = errors.New("category is not visible to the user")
	// ErrCategoryInvalidName Empty/whitespace-only name. Primary validation is handler binding.
	ErrCategoryInvalidName = errors.New("category name must not be empty")
)
