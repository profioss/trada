package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	toml "github.com/pelletier/go-toml"
)

type dataRange string

var validRange = []dataRange{"5y", "2y", "1y", "ytd", "6m", "3m", "1m", "1d"}

func (dr dataRange) validate() error {
	var empty dataRange
	if dr == empty {
		return errors.New("Range not specified")
	}

	for _, r := range validRange {
		if dr == r {
			return nil
		}
	}

	hint := make([]string, 0, len(validRange))
	for _, r := range validRange {
		hint = append(hint, string(r))
	}

	return fmt.Errorf("Range %q is not valid; use one of: %s",
		dr, strings.Join(hint, ", "))
}

func rangeLstStr(rr []dataRange) string {
	lst := make([]string, len(rr))
	for i := range rr {
		lst[i] = string(rr[i])
	}
	return strings.Join(lst, "|")
}

// Config is main configuration.
type Config struct {
	Setup            Setup
	TickerConversion map[string]string

	// cmd line flag, not part of the config file
	// updateTstData bool
	symbols []string
	verbose bool
}

// Validate checks if Config is valid.
func (c Config) Validate() error {
	switch {
	case c.Setup.Validate() != nil:
		return fmt.Errorf("Config: %s", c.Setup.Validate())
	}

	return nil
}

// Setup defines command setup.
type Setup struct {
	Range      dataRange
	BaseURL    string
	Token      string
	Timeout    time.Duration
	MaxProcs   int
	LogFile    string
	LogLevel   string
	OutputDir  string
	Watchlists []string
}

// Validate checks if Setup is valid.
func (s Setup) Validate() error {
	switch {
	case s.BaseURL == "":
		return errors.New("Setup: BaseURL is not specified")

	case s.OutputDir == "":
		return errors.New("Setup: Output Directory is not specified")

	case s.Range.validate() != nil:
		return fmt.Errorf("Setup: %s", s.Range.validate())

	case s.Timeout < 1:
		return errors.New("Setup: Timeout is set too low")

	case len(s.Watchlists) == 0:
		return errors.New("Setup: empty Watchlists definition")
	}

	return nil
}

func initConfig() (Config, error) {
	optConf := flag.String("c", "config/get-md-iex.toml", "config file")
	optLogLevel := flag.String("log-level", "", "log levels: disabled | error | warning | info | debug")
	optDirOut := flag.String("o", "", "output data directory")
	optRange := flag.String("r", "", "data range, use one of: "+rangeLstStr(validRange))
	optSymbols := flag.String("s", "", "symbols delimited by comma (ex: SPY,QQQ,DIA)")
	optTimeout := flag.Uint("t", 0, "request timeout in seconds")
	// optTstData := flag.Bool("update-test-data", false, "update test data - use with -o testdata")
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
	// conf.updateTstData = *optTstData

	conf.Setup.Timeout = time.Duration(conf.Setup.Timeout) * time.Second
	//
	// override setup from config by cmdline args
	if *optTimeout > 0 {
		conf.Setup.Timeout = time.Duration(*optTimeout) * time.Second
	}
	if *optDirOut != "" {
		conf.Setup.OutputDir = *optDirOut
	}
	if *optRange != "" {
		conf.Setup.Range = dataRange(*optRange)
	}

	conf.symbols = []string{}
	if *optSymbols != "" {
		conf.symbols = strings.Split(*optSymbols, ",")
	}

	// default log level
	// log levels: disabled | error | warning | info | debug
	if conf.Setup.LogLevel == "" {
		conf.Setup.LogLevel = "info"
	}
	if *optLogLevel != "" {
		conf.Setup.LogLevel = *optLogLevel
	}
	// disable logging if no log file was specified
	if conf.Setup.LogFile == "" {
		conf.Setup.LogLevel = "disabled"
	}

	return conf, conf.Validate()
}
