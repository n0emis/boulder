package observer

import (
	"fmt"
	"time"

	"github.com/letsencrypt/boulder/cmd"
)

var (
	// Registry is the global mapping of all `Configurer` types. Types
	// are added to this mapping on import by including a call to
	// `Register` in their `init` function
	Registry = make(map[string]Configurer)
)

// Prober is the expected interface for Prober types
type Prober interface {
	Name() string
	Kind() string
	Do(time.Duration) (bool, time.Duration)
}

// Configurer is the expected interface for Configurer types
type Configurer interface {
	UnmarshalSettings([]byte) (Configurer, error)
	Validate() error
	AsProbe() Prober
}

// Settings is exported as a temporary receiver for the `settings` field
// of the yaml config. It's always marshaled back to bytes and then
// unmarshalled into the `Configurer` specified by the `kind` field of
// the `MonConf`
type Settings map[string]interface{}

// GetProbeConf returns the probe configurer specified by name from
// `observer.Registry`
func GetProbeConf(kind string, s Settings) (Configurer, error) {
	if _, ok := Registry[kind]; ok {
		return Registry[kind], nil
	}
	return nil, fmt.Errorf("%s is not a registered probe type", kind)
}

// Register is called by every `Configurer` `init` function to add the
// caller to the global `observer.Registry` map. If the caller attempts
// to add a `Configurer` to the registry using the same name as a prior
// `Configurer` the call will exit with an error
func Register(kind string, c Configurer) {
	if _, ok := Registry[kind]; ok {
		cmd.FailOnError(
			fmt.Errorf(
				"configurer: %s has already been added", kind),
			"Error while initializing probes")
	}
	Registry[kind] = c
}
