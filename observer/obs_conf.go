package observer

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"

	"github.com/letsencrypt/boulder/cmd"
	blog "github.com/letsencrypt/boulder/log"
)

// ObsConf is exported to receive yaml configuration
type ObsConf struct {
	Syslog    cmd.SyslogConfig `yaml:"syslog"`
	DebugAddr string           `yaml:"debugaddr"`
	MonConfs  []*MonConf       `yaml:"monitors"`
}

// validateMonConfs calls the validate method for each `MonConf`. If a
// valiation error is encountered, this is appended to a slice of
// errors. If no valid `MonConf` remain, the slice of errors is returned
// along with `false`, indicating that observer should not start
func (c *ObsConf) validateMonConfs() ([]error, bool) {
	var errs []error
	for _, m := range c.MonConfs {
		err := m.validate()
		if err != nil {
			errs = append(errs, err)
		}
	}
	if len(c.MonConfs) == len(errs) {
		// all configured monitors are invalid, cannot continue
		return errs, false
	}
	return errs, true
}

// ValidateDebugAddr ensures the the `debugAddr` received by `ObsConf`
// is properly formatted and a valid port
func (c *ObsConf) ValidateDebugAddr() error {
	addrExp := regexp.MustCompile("^:([[:digit:]]{1,5})$")
	if !addrExp.MatchString(c.DebugAddr) {
		return fmt.Errorf(
			"invalid `debugaddr`, %q, not expected format", c.DebugAddr)
	}
	addrExpMatches := addrExp.FindAllStringSubmatch(c.DebugAddr, -1)
	port, _ := strconv.Atoi(addrExpMatches[0][1])
	if !(port > 0 && port < 65535) {
		return fmt.Errorf(
			"invalid `debugaddr`, %q, is not a valid port", port)
	}
	return nil
}

// validate normalizes then validates the config received the `ObsConf`
// and each of it's `MonConf`. If no valid `MonConf` remain, an error
// indicating that Observer cannot be started is returned. In all
// instances the rationale for invalidating a 'MonConf' will logged to
// stderr
func (c *ObsConf) validate(log blog.Logger) error {
	if c == nil {
		return errors.New("observer config is empty")
	}

	// validate `debugaddr`
	err := c.ValidateDebugAddr()
	if err != nil {
		return err
	}

	// operator failed to provide any monitors
	if len(c.MonConfs) == 0 {
		return errors.New("observer config is invalid, no monitors provided")
	}

	errs, ok := c.validateMonConfs()
	for mon, err := range errs {
		log.Errf("monitor %q is invalid: %s", mon, err)
	}

	if len(errs) != 0 {
		log.Errf("%d of %d monitors failed validation", len(errs), len(c.MonConfs))
	} else {
		log.Info("all monitors passed validation")
	}

	// if 0 `MonConfs` passed validation, return error
	if !ok {
		return fmt.Errorf("no valid mons, cannot continue")
	}

	return nil
}
