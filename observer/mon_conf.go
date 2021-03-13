package observer

import (
	"fmt"
	"strings"

	"github.com/letsencrypt/boulder/cmd"
	p "github.com/letsencrypt/boulder/observer/probers"
	"gopkg.in/yaml.v2"
)

// MonConf is exported to receive yaml configuration
type MonConf struct {
	Period   cmd.ConfigDuration `yaml:"period"`
	Timeout  int                `yaml:"timeout"`
	Kind     string             `yaml:"kind"`
	Settings p.Settings         `yaml:"settings"`
	Valid    bool
}

// normalize trims and lowers the string fields of `MonConf`
func (c MonConf) normalize() {
	c.Kind = strings.Trim(strings.ToLower(c.Kind), " ")
}

func (c MonConf) unmashalProbeSettings() (p.Configurer, error) {
	probeConf, err := p.GetProbeConf(c.Kind, c.Settings)
	if err != nil {
		return nil, err
	}
	s, _ := yaml.Marshal(c.Settings)
	probeConf, err = probeConf.UnmarshalSettings(s)
	if err != nil {
		return nil, err
	}
	return probeConf, nil
}

// validate normalizes and validates the received `MonConf`. If the
// `MonConf` cannot be validated, an error appropriate for end-user
// consumption is returned
func (c *MonConf) validate() error {
	c.normalize()
	probeConf, err := c.unmashalProbeSettings()
	if err != nil {
		return err
	}
	err = probeConf.Validate()
	if err != nil {
		return fmt.Errorf(
			"failed to validate: %s prober with settings: %+v due to: %w",
			c.Kind, probeConf, err)
	}
	c.Valid = true
	return nil
}

func (c MonConf) getProber() p.Prober {
	probeConf, _ := c.unmashalProbeSettings()
	return probeConf.AsProbe()
}
