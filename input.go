package main

import (
	"bufio"
	"io"
	"log"
	"os"
	"strings"

	"github.com/google/go-github/github"
	"github.com/joerx/gitmirror/pkg/gh"
)

// read list of repos by calling github api
func getReposFromGithub(ghc *github.Client, orgName string) ([]string, error) {
	log.Printf("Reading list of private repos for '%s'", orgName)

	repos, err := gh.ListPrivateReposByOrg(ghc, orgName)
	if err != nil {
		return nil, err
	}

	res := make([]string, len(repos))
	for i, r := range repos {
		res[i] = r.GetSSHURL()
	}

	return res, nil
}

// reads list of repos from given reader, e.g. a file or stdin
func getReposFromFile(r io.Reader) ([]string, error) {
	repos := []string{}
	br := bufio.NewReader(os.Stdin)

	for {
		line, err := br.ReadString('\n')
		if err != nil {
			break // assuming EOF
		}
		repos = append(repos, strings.TrimSpace(line))
	}

	return repos, nil
}
