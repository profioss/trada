package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	toml "github.com/pelletier/go-toml"
	"github.com/profioss/clog"
)

// Config is main configuration.
type Config struct {
	Setup     Setup     `toml:"setup"`
	Resources []DataSrc `toml:"resources"`

	// cmd line flags, not part of the config file
	path          string
	updateTstData bool
	verbose       bool
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
	WikiAPI   string        `toml:"wiki-api"`
	Timeout   time.Duration `toml:"timeout"`
	MaxProcs  int           `toml:"max-procs"`
	LogFile   string        `toml:"log-file"`
	LogLevel  string        `toml:"log-level"`
	OutputDir string        `toml:"output-dir"`
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

	// LogLevel and LogFile can be empty, safe defaults are used in initConfig()

	return nil
}

// DataSrc defines data sources.
type DataSrc struct {
	Name       string `toml:"name"`
	PageName   string `toml:"page-name"`
	Section    int    `toml:"section"`
	MinCnt     int    `toml:"min-cnt"`
	OutputFile string `toml:"output-file"`
}

// Validate checks if DataSrc is valid.
// TODO - match name with supported parsers
func (ds DataSrc) Validate() error {
	switch {
	case ds.Name == "":
		return errors.New("DataSrc: Name is not specified")

	case ds.PageName == "":
		return errors.New("DataSrc: PageName is not specified")

	case ds.OutputFile == "":
		return errors.New("DataSrc: OutputFile is not specified")

	case ds.Section < 1:
		return errors.New("DataSrc: Section is < 1")
	}

	_, ok := parsers[ds.Name]
	if !ok {
		names := []string{}
		for k := range parsers {
			names = append(names, k)
		}
		return fmt.Errorf("DataSrc: Name '%s' is not valid. Use one of: %s",
			ds.Name, strings.Join(names, ", "))
	}

	return nil
}

func initSettings() Config {
	cfg := Config{}

	flag.StringVar(&cfg.path, "c", "config/get-wiki-index-components.toml", "config file")
	flag.StringVar(&cfg.Setup.LogLevel, "log-level", "", "log levels: disabled | error | warning | info | debug")
	flag.StringVar(&cfg.Setup.OutputDir, "o", "", "output data directory")
	flag.BoolVar(&cfg.updateTstData, "update-test-data", false, "update test data - use with -o testdata")
	flag.BoolVar(&cfg.verbose, "v", false, "verbose mode")

	optTimeout := flag.Uint("t", 0, "request timeout in seconds")
	flag.Parse()

	if *optTimeout > 0 {
		cfg.Setup.Timeout = time.Duration(*optTimeout) * time.Second
	}

	return cfg
}

func initConfig(settings Config) (Config, error) {
	var conf Config

	if settings.path == "" {
		return conf, fmt.Errorf("empty config path")
	}
	fd, err := os.Open(settings.path)
	if err != nil {
		return conf, fmt.Errorf("config open file error: %v", err)
	}
	defer fd.Close()

	err = toml.NewDecoder(fd).Decode(&conf)
	if err != nil {
		return conf, fmt.Errorf("config %s parse error: %v", settings.path, err)
	}

	conf.verbose = settings.verbose
	conf.updateTstData = settings.updateTstData
	conf.Setup.Timeout = time.Duration(conf.Setup.Timeout) * time.Second
	//
	// override setup from config by cmdline args
	if settings.Setup.Timeout > 0 {
		conf.Setup.Timeout = settings.Setup.Timeout
	}
	if settings.Setup.OutputDir != "" {
		conf.Setup.OutputDir = settings.Setup.OutputDir
	}

	// default log level
	// log levels: disabled | error | warning | info | debug
	if conf.Setup.LogLevel == "" {
		conf.Setup.LogLevel = "info"
	}
	if settings.Setup.LogLevel != "" {
		conf.Setup.LogLevel = settings.Setup.LogLevel
	}
	_, err = clog.LevelFromString(conf.Setup.LogLevel)
	if err != nil {
		return conf, fmt.Errorf("invalid log level: %v", err)
	}
	// disable logging if no log file was specified
	if conf.Setup.LogFile == "" {
		conf.Setup.LogLevel = "disabled"
	}

	return conf, conf.Validate()
}
