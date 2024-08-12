package bbfs

import (
	"context"
	"errors"
	"io/fs"

	"github.com/myhops/bbfs/bbclient/server"
)

var (
	ErrNotBBFS = errors.New("not a bbfs FS")
)

// Tags returns the tags for the FS if it implements Tags
func Tags(f fs.FS) ([]string, error) {
	b, ok := f.(*bbFS)
	if !ok {
		return nil, ErrNotBBFS
	}

	cmd := server.GetTagsCommand{
		ProjectKey: b.projectKey,
		RepoSlug:   b.repoSlug,
	}
	tags, err := b.client.GetTags(context.Background(), &cmd)
	if err != nil {
		return nil, err
	}
	return tags, nil
}
