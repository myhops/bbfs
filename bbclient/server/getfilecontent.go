package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type GetFileContentCommand struct {
	FilePath   string
	ProjectKey string
	RepoSlug   string
	At         string
}

func (c *GetFileContentCommand) newRequestWithContext(ctx context.Context, baseURL string) (*http.Request, error) {
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

func (c *GetFileContentCommand) Validate() error {
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

func (c *GetFileContentCommand) ParseResponse(data []byte) ([]byte, error) {
	var resp struct {
		Lines []struct {
			Text string `json:"text"`
		} `json:"lines"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	var b bytes.Buffer
	for _, line := range resp.Lines {
		if _, err := fmt.Fprintln(&b, line.Text); err != nil {
			return nil, err
		}
	}
	return b.Bytes(), nil
}
