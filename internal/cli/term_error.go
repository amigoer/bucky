package cli

import "github.com/amigoer/bucky/internal/core"

func coreInvalidTerminal() error {
	return core.Invalid("Interactive credential input requires a terminal")
}
