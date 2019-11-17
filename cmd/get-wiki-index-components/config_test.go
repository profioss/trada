package main

import (
	"testing"
	"time"
)

func TestInitConfig(t *testing.T) {
	conf, err := initConfig(Config{path: "testdata/get-wiki-index-components.toml"})
	if err != nil {
		t.Fatalf("initConfig error: %v", err)
	}

	tests := []struct {
		label  string
		conf   Config
		hasErr bool
	}{
		{
			label:  "correct",
			conf:   conf, // conf parsed from config file
			hasErr: false,
		},
		{
			label: "empty Resources",
			conf: func() Config {
				c := conf
				c.Resources = []DataSrc{}
				return c
			}(),
			hasErr: true,
		},
		{
			label: "invalid Resource",
			conf: func() Config {
				c := conf
				c.Resources = append(c.Resources, DataSrc{})
				return c
			}(),
			hasErr: true,
		},
		{
			label: "invalid Setup.WikiAPI",
			conf: func() Config {
				c := conf
				c.Setup.WikiAPI = ""
				return c
			}(),
			hasErr: true,
		},
		{
			label: "invalid Setup.OutputDir",
			conf: func() Config {
				c := conf
				c.Setup.OutputDir = ""
				return c
			}(),
			hasErr: true,
		},
		{
			label: "invalid Setup.Timeout",
			conf: func() Config {
				c := conf
				c.Setup.Timeout = 0
				return c
			}(),
			hasErr: true,
		},
	}

	for _, tc := range tests {
		err = tc.conf.Validate()
		switch {
		case tc.hasErr && err == nil:
			t.Errorf("%s - should have an error", tc.label)
		case !tc.hasErr && err != nil:
			t.Errorf("%s - unexpected error: %s", tc.label, err)
		}
	}
}

func TestInitConfigOverrides(t *testing.T) {
	settings := Config{path: "testdata/get-wiki-index-components.toml"}

	settings.verbose = true
	config, err := initConfig(settings)
	switch {
	case err != nil:
		t.Fatalf("Config.verbose - unexpected error: %v", err)
	case config.verbose != settings.verbose:
		t.Errorf("Config.verbose should be %v, got %v", settings.verbose, config.verbose)
	}

	settings.updateTstData = true
	config, err = initConfig(settings)
	switch {
	case err != nil:
		t.Fatalf("Config.updateTstData - unexpected error: %v", err)
	case config.updateTstData != settings.updateTstData:
		t.Errorf("Config.updateTstData should be %v, got %v", settings.updateTstData, config.updateTstData)
	}

	var seconds float64 = 999
	settings.Setup.Timeout = time.Duration(seconds) * time.Second
	config, err = initConfig(settings)
	switch {
	case err != nil:
		t.Fatalf("Config.Setup.Timeout - unexpected error: %v", err)
	case config.Setup.Timeout != settings.Setup.Timeout:
		t.Errorf("Config.Setup.Timeout should be %v, got %v", settings.Setup.Timeout, config.Setup.Timeout)
	case config.Setup.Timeout.Seconds() != seconds:
		t.Errorf("Config.Setup.Timeout should be %v seconds, got %v", seconds, config.Setup.Timeout)
	}

	settings.Setup.OutputDir = "xy9xy9"
	config, err = initConfig(settings)
	switch {
	case err != nil:
		t.Fatalf("Config.Setup.OutputDir - unexpected error: %v", err)
	case config.Setup.OutputDir != settings.Setup.OutputDir:
		t.Errorf("Config.Setup.OutputDir should be %v, got %v", settings.Setup.OutputDir, config.Setup.OutputDir)
	}

	settings.Setup.LogLevel = "debug"
	config, err = initConfig(settings)
	switch {
	case err != nil:
		t.Fatalf("Config.Setup.LogLevel - unexpected error: %v", err)
	case config.Setup.LogLevel != settings.Setup.LogLevel:
		t.Errorf("Config.Setup.LogLevel should be %v, got %v", settings.Setup.LogLevel, config.Setup.LogLevel)
	}
}

func TestInitConfigSettings(t *testing.T) {
	tests := []struct {
		label    string
		settings Config
		hasErr   bool
	}{
		{
			label:    "correct",
			settings: Config{path: "testdata/get-wiki-index-components.toml"},
			hasErr:   false,
		},
		{
			label:    "missing Config.path",
			settings: Config{},
			hasErr:   true,
		},
		{
			label:    "invalid log level",
			settings: Config{Setup: Setup{LogLevel: "xxx"}},
			hasErr:   true,
		},
	}

	for _, tc := range tests {
		_, err := initConfig(tc.settings)
		switch {
		case tc.hasErr && err == nil:
			t.Errorf("%s - should have an error", tc.label)
		case !tc.hasErr && err != nil:
			t.Errorf("%s - unexpected error: %s", tc.label, err)
		}
	}
}

func TestDataSrcValidation(t *testing.T) {
	tdata := []struct {
		label  string
		ds     DataSrc
		hasErr bool
	}{
		{
			label: "valid",
			ds: DataSrc{
				Name:       "DJIA",
				PageName:   "Dow_Jones_Industrial_Average",
				OutputFile: "DJIA-components.csv",
				Section:    1,
				MinCnt:     25,
			},
			hasErr: false,
		},
		{
			label: "invalid Name",
			ds: DataSrc{
				Name:       "DJIAxxx",
				PageName:   "Dow_Jones_Industrial_Average",
				OutputFile: "DJIA-components.csv",
				Section:    1,
				MinCnt:     25,
			},
			hasErr: true,
		},
		{
			label: "empty Name",
			ds: DataSrc{
				Name:       "",
				PageName:   "Dow_Jones_Industrial_Average",
				OutputFile: "DJIA-components.csv",
				Section:    1,
				MinCnt:     25,
			},
			hasErr: true,
		},
		{
			label: "empty PageName",
			ds: DataSrc{
				Name:       "DJIA",
				PageName:   "",
				OutputFile: "DJIA-components.csv",
				Section:    1,
				MinCnt:     25,
			},
			hasErr: true,
		},
		{
			label: "empty OutputFile",
			ds: DataSrc{
				Name:       "DJIA",
				PageName:   "Dow_Jones_Industrial_Average",
				OutputFile: "",
				Section:    1,
				MinCnt:     25,
			},
			hasErr: true,
		},
		{
			label: "invalid Section",
			ds: DataSrc{
				Name:       "DJIA",
				PageName:   "Dow_Jones_Industrial_Average",
				OutputFile: "DJIA-components.csv",
				Section:    0,
				MinCnt:     25,
			},
			hasErr: true,
		},
	}

	for _, tc := range tdata {
		err := tc.ds.Validate()
		switch {
		case err != nil && !tc.hasErr:
			t.Errorf("%s - unexpected error: %v", tc.label, err)
		case err == nil && tc.hasErr:
			t.Errorf("%s - should have error", tc.label)
		}
	}
}
