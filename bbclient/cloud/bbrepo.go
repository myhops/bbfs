package cloud

import (
	"context"
	"net/url"
)

type BitbucketRepo struct {
	Client     *Client
	ProjectKey string
	RepoSlug   string
}

// GetContent implements server.BitbucketRepository.
func (r *BitbucketRepo) GetContent(ctx context.Context, component string, version string, filePath string) ([]byte, error) {
	path, err := url.JoinPath("", component, filePath)
	if err != nil {
		return nil, err
	}
	at := "refs/tags/" + component + "/" + version
	cmd := &GetFileContentCommand{
		ProjectKey: r.ProjectKey,
		RepoSlug:   r.RepoSlug,
		FilePath:   path,
		At:         at,
	}
	return r.Client.GetFileContent(ctx, cmd)
}

// GetTags implements server.BitbucketRepository.
func (r *BitbucketRepo) GetTags(ctx context.Context) ([]string, error) {
	cmd := &GetTagsCommand{
		ProjectKey: r.ProjectKey,
		RepoSlug:   r.RepoSlug,
	}
	return r.Client.GetTags(ctx, cmd)
}

// var _ server.BitbucketRepository = &BitbucketRepo{}
