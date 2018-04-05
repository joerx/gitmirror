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
	"github.com/joerx/gitmirror/pkg/ccm"
	"github.com/joerx/gitmirror/pkg/gitmirror"
	"github.com/urfave/cli"
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
			Name:  "aws-region",
			Value: "ap-southeast-1",
			Usage: "AWS region to create archived repos in",
		},
	}

	app.Action = actionMigrateRepos

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func actionMigrateRepos(c *cli.Context) error {
	var repos []string
	var err error

	origin := c.String("origin")
	region := c.String("aws-region")

	sess := session.Must(session.NewSession(&aws.Config{
		Region: &region,
	}))
	cc := codecommit.New(sess)

	if len(origin) == 0 {
		repos, err = getListOfRepos(os.Stdin)
		if err != nil {
			return err
		}
	} else {
		repos = []string{origin}
	}

	wd, err := createWorkDir("work")
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

// reads list of repos from given reader, e.g. a file or stdin
func getListOfRepos(r io.Reader) ([]string, error) {
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
