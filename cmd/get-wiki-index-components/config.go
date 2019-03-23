package main

import (
	"errors"
	"flag"
	"os"
	"time"

	toml "github.com/pelletier/go-toml"
	"github.com/profioss/clog"
)

// Config is main configuration.
type Config struct {
	Setup     Setup
	Resources []DataSrc

	// not part of the config file
	log     clog.Logger
	verbose bool
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

// DataSrc defines data sources.
type DataSrc struct {
	Name       string
	PageName   string
	Section    int
	MinCnt     int
	OutputFile string
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

	return conf, checkConfig(conf)
}

func checkConfig(c Config) error {
	switch {
	case c.Setup.WikiAPI == "":
		return errors.New("checkConfig: WikiAPI is not specified")

	case c.Setup.OutputDir == "":
		return errors.New("checkConfig: Output Directory is not specified")

	case c.Setup.Timeout < 1:
		return errors.New("checkConfig: Timeout is set too low")

	case len(c.Resources) == 0:
		return errors.New("checkConfig: Empty Resources definition - nothing to do")
	}

	return nil
}
