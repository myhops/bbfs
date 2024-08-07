package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// GetTagsCommand is the command to retrieve all tags from the repository.
type GetTagsCommand struct {
	ProjectKey string
	RepoSlug   string
	OrderBy    string
}

func (c *GetTagsCommand) Validate() error {
	if c.ProjectKey == "" {
		return fmt.Errorf("ProjectKey is missing")
	}
	if c.RepoSlug == "" {
		return fmt.Errorf("RepoSlug is missing")
	}
	return nil
}

func (c *GetTagsCommand) newRequestWithContext(ctx context.Context, baseURL string) (*http.Request, error) {
	u, err := url.Parse(fmt.Sprintf("%s/projects/%s/repos/%s/tags", baseURL, c.ProjectKey, c.RepoSlug))
	if err != nil {
		return nil, err
	}
	var vals url.Values
	addValue(vals, "orderBy", c.OrderBy)
	u.RawQuery = vals.Encode()
	us := u.String()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, us, nil)
	if err != nil {
		return nil, err
	}
	return req, nil
}

func (c *GetTagsCommand) ParseResponse(data []byte) ([]string, error) {
	var resp struct {
		Values []struct {
			ID        string `json:"id"`
			DisplayID string `json:"displayId"`
		} `json:"values"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	var tags = make([]string, 0, len(resp.Values))
	for _, tag := range resp.Values {
		tags = append(tags, tag.DisplayID)
	}
	return tags, nil
}
