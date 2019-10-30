package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	toml "github.com/pelletier/go-toml"
)

type dataRange string

// NOTE max is upto 15 years
var validRange = []dataRange{"max", "5y", "2y", "1y", "ytd", "6m", "3m", "1m", "1d"}

func newDataRange(r string) dataRange {
	input := strings.TrimSpace(r)
	input = strings.ToLower(input)
	return dataRange(input)
}

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

// since converts dataRange to corresponding time.
func (dr dataRange) since() (time.Time, error) {
	output := time.Now().UTC().Truncate(time.Hour * 24)

	if dr.validate() != nil {
		return output, dr.validate()
	}

	drs := string(dr)
	switch drs {
	case "max":
		// beginning of the year and 15 years ago
		output = time.Date(output.Year(), 1, 1, 0, 0, 0, 0, time.UTC).AddDate(-15, 0, 0)
		return output, nil
	case "ytd":
		output = time.Date(output.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
		return output, nil
	}

	tunit := string(drs[1])
	n, err := strconv.ParseInt(string(drs[0]), 10, 64)
	if err != nil {
		return output, fmt.Errorf("unable to parse %q as integer: %v", drs[0], err)
	}
	num := int(n)

	switch tunit {
	case "y":
		output = output.AddDate(-num, 0, 0)
	case "m":
		output = output.AddDate(0, -num, 0)
	case "d":
		output = output.AddDate(0, 0, -num)
	}

	return output, nil
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

	case len(c.Setup.Watchlists) == 0 && len(c.symbols) == 0:
		return errors.New("Setup: empty Watchlists definition nor -s flag used")
	}

	return nil
}

// Setup defines command setup.
type Setup struct {
	Range      dataRange
	BaseURL    string
	Exchange   string
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

	case s.Exchange == "":
		return errors.New("Setup: Exchange is not specified; see: https://cryptowat.ch/exchanges")

	case s.OutputDir == "":
		return errors.New("Setup: Output Directory is not specified")

	case s.Range.validate() != nil:
		return fmt.Errorf("Setup: %s", s.Range.validate())

	case s.Timeout < 1:
		return errors.New("Setup: Timeout is set too low")
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
		conf.Setup.Range = newDataRange(*optRange)
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
