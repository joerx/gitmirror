package gh

import (
	"context"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

// NewFromToken creates a `Client` using the given token for authentication
func NewFromToken(token string) *github.Client {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)

	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	return client
}

// ListPrivateReposByOrg will list private git repos by organisation
func ListPrivateReposByOrg(cl *github.Client, orgName string) ([]*github.Repository, error) {
	allRepos := []*github.Repository{}
	opt := &github.RepositoryListByOrgOptions{Type: "private"}

	for {
		repos, resp, err := cl.Repositories.ListByOrg(context.Background(), orgName, opt)
		if err != nil {
			return nil, err
		}
		allRepos = append(allRepos, repos...)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return allRepos, nil
}
