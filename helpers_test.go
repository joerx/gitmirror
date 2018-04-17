package main

import "testing"

type parseRemoteTest struct {
	input string
	name  string
	org   string
}

func TestParseRemoteURL(t *testing.T) {
	testCases := []parseRemoteTest{
		{
			input: "git@github.com:honestbee/design.honestbee.com.git",
			name:  "design.honestbee.com",
			org:   "honestbee",
		},
		{
			input: "git@github.com:honestbee/HB-CMS.git",
			name:  "HB-CMS",
			org:   "honestbee",
		},
	}

	for _, tc := range testCases {
		org, name, err := parseRemoteURL(tc.input)
		if err != nil {
			t.Error(err)
			continue
		}
		if org != tc.org {
			t.Errorf("Expected org to be %s but was %s for input %s", tc.org, org, tc.input)
		}
		if name != tc.name {
			t.Errorf("Expected name t be %s but was %s for input %s", tc.name, name, tc.input)
		}
	}
}
