package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	toml "github.com/pelletier/go-toml"
)

// Config is main configuration.
type Config struct {
	Setup     Setup
	Resources []DataSrc

	// cmd line flag, not part of the config file
	verbose bool
}

// Validate checks if Config is valid.
func (c Config) Validate() error {
	switch {
	case c.Setup.Validate() != nil:
		return fmt.Errorf("Config: %s", c.Setup.Validate())

	case len(c.Resources) == 0:
		return errors.New("Config: empty Resources definition")
	}

	for _, r := range c.Resources {
		if r.Validate() != nil {
			return fmt.Errorf("Config: %s", r.Validate())
		}
	}

	return nil
}

// Setup defines command setup.
type Setup struct {
	WikiAPI   string
	Timeout   time.Duration
	MaxProcs  int
	LogFile   string
	LogLevel  string
	OutputDir string
}

// Validate checks if Setup is valid.
func (s Setup) Validate() error {
	switch {
	case s.WikiAPI == "":
		return errors.New("Setup: WikiAPI is not specified")

	case s.OutputDir == "":
		return errors.New("Setup: Output Directory is not specified")

	case s.Timeout < 1:
		return errors.New("Setup: Timeout is set too low")
	}

	return nil
}

// DataSrc defines data sources.
type DataSrc struct {
	Name       string
	PageName   string
	Section    int
	MinCnt     int
	OutputFile string
}

// Validate checks if DataSrc is valid.
// TODO - match name with supported parsers
func (ds DataSrc) Validate() error {
	switch {
	case ds.Name == "":
		return errors.New("DataSrc: Name is not specified")

	case ds.PageName == "":
		return errors.New("DataSrc: PageName is not specified")

	case ds.Section < 1:
		return errors.New("DataSrc: Section is < 1")

	case ds.OutputFile == "":
		return errors.New("DataSrc: OutputFile is not specified")
	}

	return nil
}

func initConfig() (Config, error) {
	optConf := flag.String("c", "config/get-wiki-index-components.toml", "config file")
	optDirOut := flag.String("o", "", "output data directory")
	optTimeout := flag.Uint("t", 0, "request timeout in seconds")
	optVerb := flag.Bool("v", false, "verbose mode")
	flag.Parse()

	var conf Config
	fd, err := os.Open(*optConf)
	if err != nil {
		return conf, err
	}
	defer fd.Close()

	err = toml.NewDecoder(fd).Decode(&conf)
	if err != nil {
		return conf, err
	}

	conf.verbose = *optVerb

	conf.Setup.Timeout = time.Duration(conf.Setup.Timeout) * time.Second
	if *optTimeout > 0 {
		conf.Setup.Timeout = time.Duration(*optTimeout) * time.Second
	}
	// override setup from config by cmdline args
	if *optDirOut != "" {
		conf.Setup.OutputDir = *optDirOut
	}

	// default log level
	// log levels: disabled | error | warning | info | debug
	if conf.Setup.LogLevel == "" {
		conf.Setup.LogLevel = "info"
	}
	// disable logging if no log file was specified
	if conf.Setup.LogFile == "" {
		conf.Setup.LogLevel = "disabled"
	}

	return conf, conf.Validate()
}
