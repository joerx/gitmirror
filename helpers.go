package main

import (
	"errors"
	"regexp"
)

var remoteRE = regexp.MustCompile(`git@github.com:([-_\w\.]*)/([-_\w\.]*).git`)

// parseRemoteURL parses repo name and owner from its github clone URL
// this is a lot faster than having to look up the repo matadata via API
func parseRemoteURL(url string) (string, string, error) {
	m := remoteRE.FindStringSubmatch(url)
	if len(m) == 0 {
		return "", "", errors.New("Failed to parse remote " + url)
	}
	return m[1], m[2], nil
}

func indentStrings(s []string) []string {
	res := make([]string, len(s))
	for i, str := range s {
		res[i] = "  " + str
	}
	return res
}
