package gitmirror

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
)

// RemoteConfigure sets up the given git remote for the git repo given by `dest`. It will inject the config
// directly with custom refspec to avoid pulling some refs that can't be mirrored.
// (See http://christoph.ruegg.name/blog/git-howto-mirror-a-github-repository-without-pull-refs.html)
func RemoteConfigure(dest string, cloneURL string, remoteName string) error {
	p := path.Join(dest, "config")

	f1, err := os.OpenFile(p, os.O_RDONLY, 0655)
	if err != nil {
		return err
	}

	defer f1.Close()

	r := bufio.NewReader(f1)
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			break //EOF
		}
		if m, _ := regexp.MatchString(`\[remote "`+remoteName+`"\]`, line); m {
			log.Printf("Remote '%s' already configured", remoteName)
			return nil
		}
	}

	f2, err := os.OpenFile(p, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	defer f2.Close()

	tpl := `[remote "%s"]
	url = %s
	fetch = +refs/heads/*:refs/heads/*
	fetch = +refs/tags/*:refs/tags/*`

	_, err = fmt.Fprintf(f2, tpl+"\n", remoteName, cloneURL)
	if err != nil {
		return err
	}

	return nil
}

// Push will execute `git push --mirror $remoteName` for the repo at `dest`
func Push(dest string, remoteName string) error {
	log.Printf("Pushing to %s", remoteName)
	out, err := cmdRun("git", "-C", dest, "push", "--mirror", remoteName)
	if err != nil {
		return errors.New(out)
	}
	log.Printf("Pushed, %s", out)
	return nil
}

// CloneOrUpdate clone repo at `url` or do a `remote update` if it already exists
func CloneOrUpdate(url string, dest string) error {
	err := Clone(url, dest)
	if e, ok := err.(gitError); ok && e.code == errAlreadyCloned {
		return RemoteUpdate(dest, "origin")
	}
	return err
}

// Clone will clone the repos at `url` into the path given by `dest` using the `--mirror` flag,
// effectively creating a bare copy
func Clone(url string, dest string) error {
	if _, err := os.Stat(dest); err == nil {
		log.Printf("Working copy at %s exists, skipping", dest)
		return newErrorf(errAlreadyCloned, "Working copy at %s already exists", dest) // repo already exists, skip
	}

	log.Printf("Cloning %s into %s", url, dest)
	if out, err := cmdRun("git", "clone", "--mirror", url, dest); err != nil {
		return errors.New(out)
	}
	return nil
}

// RemoteUpdate runs `git remote update --prune` for given `remoteName` on the repo at path `dest`
func RemoteUpdate(dest string, remoteName string) error {
	log.Printf("Updating remote %s", remoteName)
	out, err := cmdRun("git", "-C", dest, "remote", "update", "--prune", remoteName)
	if err != nil {
		return errors.New(out)
	}
	return nil
}

// RepoName will extract the repository name part from its URL
func RepoName(url string) string {
	return strings.Replace(path.Base(url), path.Ext(url), "", 1)
}

// Run command on shell
func cmdRun(cmd string, args ...string) (string, error) {
	log.Printf("Run %s %s", cmd, strings.Join(args, " "))
	c := exec.Command(cmd, args...)
	out, err := c.CombinedOutput()
	return string(out), err
}
