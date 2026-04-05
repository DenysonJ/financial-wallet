package vo

import pkgvo "github.com/DenysonJ/financial-wallet/pkg/vo"

// ID is an alias for the shared ID value object in pkg/vo.
// All domains should import pkg/vo directly for new code.
type ID = pkgvo.ID

// NewID generates a new UUID v7 identifier.
var NewID = pkgvo.NewID

// ParseID validates and parses a UUID v7 string into an ID.
var ParseID = pkgvo.ParseID
