package main

import "testing"

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
				Name:       "DJIAxxx", // invalid Name
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
