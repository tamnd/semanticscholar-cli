package cli

import (
	"errors"

	"github.com/tamnd/semanticscholar-cli/semanticscholar"
)

// isNotFound reports whether err wraps semanticscholar.ErrNotFound.
func isNotFound(err error) bool {
	return errors.Is(err, semanticscholar.ErrNotFound)
}
