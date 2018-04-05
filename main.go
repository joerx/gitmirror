package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/codecommit"
	"github.com/urfave/cli"

	"github.com/joerx/gitmirror/pkg/ccm"
	"github.com/joerx/gitmirror/pkg/gh"
	"github.com/joerx/gitmirror/pkg/gitmirror"
)

func main() {

	app := cli.NewApp()
	app.Name = "Git batch mirroring tool"
	app.Usage = "Clone and mirror a list of git repos from Github to AWS CodeCommit"

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "origin",
			Usage: "Origin repo, if not given via stdin",
			Value: "",
		},
		cli.StringFlag{
			Name:   "aws-region",
			Value:  "ap-southeast-1",
			Usage:  "AWS region to create archived repos in",
			EnvVar: "AWS_REGION",
		},
		cli.StringFlag{
			Name:   "workdir",
			Value:  "work",
			Usage:  "Directory used for temporary working copies managed by this tool",
			EnvVar: "GITMIRROR_WORKDIR",
		},
		cli.StringFlag{
			Name:   "gh-org",
			Value:  "",
			Usage:  "Org name on github to mirror repos for",
			EnvVar: "GITMIRROR_GH_ORG",
		},
		cli.StringFlag{
			Name:   "gh-token",
			Value:  "",
			Usage:  "Github access token to list repos from github",
			EnvVar: "GITMIRROR_GH_TOKEN",
		},
	}

	app.Action = actionMigrateRepos

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func actionMigrateRepos(c *cli.Context) error {
	region := c.String("aws-region")
	workdir := c.String("workdir")

	log.Printf("Using AWS region %s", region)
	log.Printf("Work dir is %s", workdir)

	repos, err := getRepos(c)
	if err != nil {
		return err
	}

	log.Printf("Found %d repos to mirror", len(repos))
	fmt.Println(repos)

	sess := session.Must(session.NewSession(&aws.Config{
		Region: &region,
	}))
	cc := codecommit.New(sess)

	wd, err := createWorkDir(workdir)
	if err != nil {
		return err
	}

	for _, repo := range repos {
		if err := archiveRepo(repo, wd, cc); err != nil {
			return err
		}
	}

	return nil
}

func createWorkDir(name string) (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	fp := path.Join(wd, name)

	if err := os.MkdirAll(fp, 0755); err != nil {
		return "", err
	}

	return fp, nil
}

func archiveRepo(repo string, workDir string, cc *codecommit.CodeCommit) error {
	log.Printf("Archiving %s", repo)

	name := gitmirror.RepoName(repo) //GitRepoName(repo)
	dest := path.Join(workDir, name)
	desc := fmt.Sprintf("Archived version of %s", repo)
	remoteName := "archive"

	if err := gitmirror.CloneOrUpdate(repo, dest); err != nil {
		return err
	}

	meta, err := ccm.RepoEnsure(name, desc, cc)
	if err != nil {
		return err
	}

	cloneURL := *meta.CloneUrlSsh
	log.Printf("Created repo with clone url %s", cloneURL)

	if err := gitmirror.RemoteConfigure(dest, cloneURL, remoteName); err != nil {
		return err
	}

	if err := gitmirror.Push(dest, remoteName); err != nil {
		return err
	}

	return nil
}

func getRepos(c *cli.Context) ([]string, error) {
	// origin param has highest precedence
	origin := c.String("origin")
	if len(origin) != 0 {
		return []string{origin}, nil
	}

	// if gh org name is given, try to get list from github
	ghOrg := c.String("gh-org")
	ghToken := c.String("gh-token")
	if len(ghOrg) != 0 {
		return getReposFromGithub(ghToken, ghOrg)
	}

	// else try to read from stdin, list may be empty
	return getReposFromFile(os.Stdin)
}

// read list of repos by calling github api
func getReposFromGithub(ghToken string, orgName string) ([]string, error) {
	log.Printf("Reading list of private repos for '%s'", orgName)

	ghc, err := gh.NewFromToken(ghToken)
	if err != nil {
		return nil, err
	}

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
