package main

import (
	"flag"
	"io/ioutil"

	"github.com/letsencrypt/boulder/cmd"
	"github.com/letsencrypt/boulder/observer"
	"gopkg.in/yaml.v2"
)

func main() {
	configPath := flag.String(
		"config", "config.yml", "Path to boulder-observer configuration file")
	flag.Parse()

	configYAML, err := ioutil.ReadFile(*configPath)
	cmd.FailOnError(err, "failed to read config file")

	// parse YAML config
	var config observer.ObsConf
	err = yaml.Unmarshal(configYAML, &config)
	if err != nil {
		cmd.FailOnError(err, "failed to parse YAML config")
	}

	err = config.ValidateDebugAddr()
	if err != nil {
		cmd.FailOnError(err, "config")
	}

	// start monitoring and logging
	prom, logger := cmd.StatsAndLogging(config.Syslog, config.DebugAddr)
	defer logger.AuditPanic()
	logger.Info(cmd.VersionString())

	// validate config and create an observer object
	observer, err := observer.New(config, logger, prom)
	if err != nil {
		cmd.FailOnError(err, "config failed validation")
	}

	// start daemon
	logger.Infof("Initializing boulder-observer daemon from config: %s", *configPath)
	logger.Debugf("Using config: %+v", config)
	observer.Start()
}
