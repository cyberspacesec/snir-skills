package runner

import (
	"fmt"

	"github.com/cyberspacesec/snir-skills/pkg/models"
)

// ChromeNotFoundError signals that chrome is not available
type ChromeNotFoundError struct {
	Err error
}

func (e ChromeNotFoundError) Error() string {
	return fmt.Sprintf("chrome not found: %v", e.Err)
}

// Driver is the interface browser drivers will implement.
type Driver interface {
	Witness(target string, opts *Options) (*models.Result, error)
	Close()
}
