package ccm

import (
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/codecommit"
)

// ccIsErr will check if given error matches given error code. Returns false if `err` is nil
func ccIsErr(err error, awsErrCode string) bool {
	if err == nil {
		return false
	}
	if _, ok := err.(awserr.RequestFailure); !ok {
		return false
	}
	return err.(awserr.RequestFailure).Code() == awsErrCode
}

// RepoEnsure ensures repos with given name exists, creating it if necessary. Returns `RepositoryMetadata` of
// the new/existing repo.
func RepoEnsure(name string, desc string, cc *codecommit.CodeCommit) (*codecommit.RepositoryMetadata, error) {
	createRepoOut, err := cc.CreateRepository(&codecommit.CreateRepositoryInput{
		RepositoryName:        &name,
		RepositoryDescription: &desc,
	})

	if err != nil {
		if ccIsErr(err, codecommit.ErrCodeRepositoryNameExistsException) {
			return RepoGet(name, cc)
		}
	}

	return createRepoOut.RepositoryMetadata, nil
}

// RepoGet will get metadata for given CodeCommit repo
func RepoGet(name string, cc *codecommit.CodeCommit) (*codecommit.RepositoryMetadata, error) {
	getRepoOut, err := cc.GetRepository(&codecommit.GetRepositoryInput{
		RepositoryName: &name,
	})
	if err != nil {
		return nil, err
	}
	return getRepoOut.RepositoryMetadata, nil
}
