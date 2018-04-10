package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/codecommit"
	"github.com/google/go-github/github"
	"github.com/urfave/cli"

	"github.com/joerx/gitmirror/pkg/ccm"
	"github.com/joerx/gitmirror/pkg/gh"
	"github.com/joerx/gitmirror/pkg/gitmirror"
)

type stats struct {
	total   int
	success int
	skipped []string
}

func (s *stats) skip(name string) {
	s.skipped = append(s.skipped, name)
}

func main() {
	app := cli.NewApp()
	app.Name = "Git batch mirroring tool"
	app.Usage = "Clone and mirror a list of git repos from Github to AWS CodeCommit"

	app.Commands = []cli.Command{
		{
			Name:   "mirror",
			Action: actionMigrateRepos,
			Usage:  "Mirror repositories",
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "destroy",
					Usage: "If set, delete source repo after migrating",
				},
				cli.StringFlag{
					Name:  "origin",
					Usage: "Origin repo, if omitted will read list from stdin",
					Value: "",
				},
			},
		},
		{
			Name:   "restore",
			Action: actionRestoreRepos,
			Usage:  "Restore repositories",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "origin",
					Usage: "Origin repo, if omitted will read list from stdin",
					Value: "",
				},
			},
		},
	}

	app.Flags = []cli.Flag{
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

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

type cliOpts struct {
	region  string
	workdir string
	destroy bool
	origin  string
	ghOrg   string
	ghToken string
}

func parseFlags(c *cli.Context) (*cliOpts, error) {
	opts := &cliOpts{
		region:  c.GlobalString("aws-region"),
		workdir: c.GlobalString("workdir"),
		ghOrg:   c.GlobalString("gh-org"),
		ghToken: c.GlobalString("gh-token"),
		destroy: c.Bool("destroy"),
		origin:  c.String("origin"),
	}

	if len(opts.ghToken) == 0 {
		return nil, errors.New("Github token is required")
	}

	if len(opts.region) == 0 {
		return nil, errors.New("AWS region is required")
	}

	return opts, nil
}

func actionRestoreRepos(c *cli.Context) error {
	opts, err := parseFlags(c)
	if err != nil {
		return err
	}

	log.Printf("Using AWS region %s", opts.region)
	log.Printf("Work dir is %s", opts.workdir)

	ghc := gh.NewFromToken(opts.ghToken)

	// aws codecommit client
	sess := session.Must(session.NewSession(&aws.Config{Region: &opts.region}))
	cc := codecommit.New(sess)

	var repos []string

	switch {
	case len(opts.origin) > 0:
		repos = []string{opts.origin}
	default:
		repos, err = getReposFromFile(os.Stdin)
	}

	rOpts := &restoreReposOpts{
		repos:   repos,
		workdir: opts.workdir,
	}

	if err = restoreRepos(ghc, cc, rOpts); err != nil {
		return err
	}

	return nil
}

type restoreReposOpts struct {
	repos   []string
	workdir string
	ghOrg   string
}

func restoreRepos(ghc *github.Client, cc *codecommit.CodeCommit, opts *restoreReposOpts) error {

	wd, err := createWorkDir(opts.workdir)
	if err != nil {
		return err
	}

	for _, url := range opts.repos {
		org, name, err := parseRemoteURL(url)
		if err != nil {
			return err
		}
		if err := restoreRepo(name, org, wd, cc, ghc); err != nil {
			log.Print(err)
			continue
		}
	}

	return nil
}

func ensureRepo(name string, org string, ghc *github.Client) (*github.Repository, error) {
	ctx := context.Background()
	repo, _, err := ghc.Repositories.Get(ctx, org, name)

	if err != nil {
		if ghe, ok := err.(*github.ErrorResponse); ok && ghe.Response.StatusCode == 404 {
			log.Printf("Repo %s/%s does not exist, creating it", org, name)
			return createRepo(name, org, ghc)
		}
		return nil, err
	}

	return repo, nil
}

func createRepo(name string, org string, ghc *github.Client) (*github.Repository, error) {
	ctx := context.Background()
	repo, _, err := ghc.Repositories.Create(ctx, org, &github.Repository{
		Name: &name,
	})
	return repo, err
}

// restoreRepo will restore the repo from archive
func restoreRepo(name string, org string, wd string, cc *codecommit.CodeCommit, ghc *github.Client) error {

	repo, err := ensureRepo(name, org, ghc)
	if err != nil {
		return err
	}

	pushURL := *repo.SSHURL
	log.Printf("Push URL for %s/%s is %s", org, name, pushURL)

	out, err := cc.GetRepository(&codecommit.GetRepositoryInput{
		RepositoryName: &name,
	})

	if err != nil {
		return err
	}

	cloneURL := *out.RepositoryMetadata.CloneUrlSsh
	dest := path.Join(wd, name)

	if err := gitmirror.CloneOrUpdate(cloneURL, dest); err != nil {
		return err
	}

	if err := gitmirror.RemoteConfigure(dest, pushURL, "github"); err != nil {
		return err
	}

	if err := gitmirror.Push(dest, "github"); err != nil {
		return err
	}

	return nil
}

func actionMigrateRepos(c *cli.Context) error {

	opts, err := parseFlags(c)
	if err != nil {
		return err
	}

	log.Printf("Using AWS region %s", opts.region)
	log.Printf("Work dir is %s", opts.workdir)

	var repos []string
	ghc := gh.NewFromToken(opts.ghToken)

	// aws codecommit client
	sess := session.Must(session.NewSession(&aws.Config{Region: &opts.region}))
	cc := codecommit.New(sess)

	switch {
	case len(opts.origin) > 0:
		repos = []string{opts.origin}
	case len(opts.ghOrg) > 0:
		repos, err = getReposFromGithub(ghc, opts.ghOrg)
	default:
		repos, err = getReposFromFile(os.Stdin)
	}

	if err != nil {
		return err
	}

	log.Printf("Found %d repos to mirror", len(repos))
	s, err := mirrorRepos(ghc, cc, &mirrorReposOpts{
		repos:   repos,
		destroy: opts.destroy,
		workdir: opts.workdir,
	})

	if err != nil {
		return err
	}

	log.Printf("Found %d repos to mirror, %d successful, %d skipped", s.total, s.success, len(s.skipped))
	log.Printf("Repos with errors:\n%s", strings.Join(indentStrings(s.skipped), "\n"))

	return nil
}

type mirrorReposOpts struct {
	repos   []string
	destroy bool
	workdir string
}

func mirrorRepos(gh *github.Client, cc *codecommit.CodeCommit, opts *mirrorReposOpts) (*stats, error) {

	s := &stats{total: len(opts.repos)}

	wd, err := createWorkDir(opts.workdir)
	if err != nil {
		return nil, err
	}

	for _, url := range opts.repos {
		if err := mirrorRepo(url, wd, cc); err != nil {
			log.Print(err)
			s.skip(url)
			continue
		}

		if opts.destroy {
			owner, name, err := parseRemoteURL(url)
			if err != nil {
				return nil, err
			}
			if err := deleteRepo(gh, owner, name); err != nil {
				log.Print(err)
				s.skip(url)
				continue
			}
			log.Printf("Deleted repo at url %s", url)
		}
		s.success++
	}

	return s, nil
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

func mirrorRepo(repo string, workDir string, cc *codecommit.CodeCommit) error {
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

func deleteRepo(ghc *github.Client, owner, name string) error {
	ctx := context.Background()
	if _, err := ghc.Repositories.Delete(ctx, owner, name); err != nil {
		return err
	}
	return nil
}
