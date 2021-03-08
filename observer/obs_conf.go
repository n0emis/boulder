package observer

import (
	"errors"
	"fmt"

	"github.com/letsencrypt/boulder/cmd"
	blog "github.com/letsencrypt/boulder/log"
)

// ObsConf is exported to receive yaml configuration
type ObsConf struct {
	Syslog    cmd.SyslogConfig `yaml:"syslog"`
	DebugAddr string           `yaml:"debugaddr"`
	MonConfs  []*MonConf       `yaml:"monitors"`
}

// validateMonConfs calls the validate method for each of the received
// `MonConf` objects. If an error is encountered, this is appended to a
// slice of errors. If no `MonConf` remain, the list of errors is
// returned along with `false`, indicating there are 0 valid `MonConf`.
// Otherwise, the list of errors is returned to be presented to the
// end-user, and true is returned to indicate that there is at least one
// valid `MonConf`
func (c *ObsConf) validateMonConfs() ([]error, bool) {
	var validationErrs []error
	for _, m := range c.MonConfs {
		err := m.validate()
		if err != nil {
			validationErrs = append(validationErrs, err)
		}
	}

	// all configured monitors are invalid, cannot continue
	if len(c.MonConfs) == len(validationErrs) {
		return validationErrs, false
	}
	return validationErrs, true
}

// validate normalizes and validates the observer config as well as each
// `MonConf`. If no valid `MonConf` remain, an error indicating that
// Observer cannot be started is returned. In all instances the the
// rationale for invalidating a 'MonConf' will logged to stderr
func (c *ObsConf) validate(log blog.Logger) error {
	if c == nil {
		return errors.New("observer config is empty")
	}

	if len(c.MonConfs) == 0 {
		return errors.New("observer config is invalid, 0 monitors configured")
	}

	logErrs := func(errs []error, lenMons int) {
		log.Errf("%d of %d monitors failed validation", len(errs), lenMons)
		for _, err := range errs {
			log.Errf("invalid monitor: %s", err)
		}
	}

	errs, ok := c.validateMonConfs()

	// if no valid `MonConf` remain, log validation errors, return error
	if len(errs) != 0 && !ok {
		logErrs(errs, len(c.MonConfs))
		return fmt.Errorf("no valid mons, cannot continue")
	}

	// if at least one valid `MonConf` remains, only log validation
	// errors
	if len(errs) != 0 && ok {
		logErrs(errs, len(c.MonConfs))
	}
	return nil
}
