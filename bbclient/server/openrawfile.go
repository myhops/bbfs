package server

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

type OpenRawFileCommand struct {
	FilePath   string
	ProjectKey string
	RepoSlug   string
	At         string
}

func (c *OpenRawFileCommand) newRequestWithContext(ctx context.Context, baseURL string) (*http.Request, error) {
	u, err := url.Parse(fmt.Sprintf("%s/projects/%s/repos/%s/raw/%s", baseURL, c.ProjectKey, c.RepoSlug, c.FilePath))
	if err != nil {
		return nil, err
	}
	vals := url.Values{}
	addValue(vals, "at", c.At)
	u.RawQuery = vals.Encode()

	us := u.String()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, us, nil)
	if err != nil {
		return nil, err
	}
	return req, nil
}

func (c *OpenRawFileCommand) Validate() error {
	if c.ProjectKey == "" {
		return fmt.Errorf("ProjectKey is missing")
	}
	if c.RepoSlug == "" {
		return fmt.Errorf("RepoSlug is missing")
	}
	if c.FilePath == "" {
		return fmt.Errorf("FilePath is missing")
	}
	return nil
}

