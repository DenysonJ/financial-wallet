package interfaces

import (
	"io"

	"github.com/DenysonJ/financial-wallet/pkg/ofx"
)

// OFXParser is the use-case-level contract for parsing OFX files. Decoupling
// the use case from a concrete parser keeps the use case independently
// testable (stub parsers can feed canned ParseResults without needing full
// valid OFX strings) and leaves room for alternative parser backends.
type OFXParser interface {
	Parse(r io.Reader) (*ofx.ParseResult, error)
}
