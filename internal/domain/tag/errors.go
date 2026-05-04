package tag

import "errors"

// Domain errors for Tag.
var (
	ErrTagNotFound  = errors.New("tag not found")
	ErrTagDuplicate = errors.New("tag already exists")
	ErrTagReadOnly  = errors.New("system tag is read-only")
	// ErrTagNotVisible Returned instead of NotFound to avoid cross-user existence oracle.
	ErrTagNotVisible = errors.New("tag is not visible to the user")
	// ErrTagLimitExceeded More than MaxTagsPerStatement unique tags requested.
	ErrTagLimitExceeded = errors.New("tag limit per statement exceeded")
	// ErrTagInvalidName Empty/whitespace-only name. Primary validation is handler binding.
	ErrTagInvalidName = errors.New("tag name must not be empty")
)
