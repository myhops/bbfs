package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

const (
	TagTypeTag    = "TAG"
	TagTypeBranch = "BRANCH"
)

// GetTagsCommand is the command to retrieve all tags from the repository.
type GetTagsCommand struct {
	ProjectKey string
	RepoSlug   string
	OrderBy    string
	Start      int
	Limit      int
}

type Tag struct {
	Name     string
	CommitID string
	Type     string
}

type GetTagsResponse struct {
	IsLastPage    bool
	Limit         int
	NextPageStart int
	Size          int
	Start         int
	Tags          []*Tag
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
	vals := url.Values{}
	addValue(vals, "orderBy", c.OrderBy)
	addValue(vals, "start", strconv.Itoa(c.Start))
	addValue(vals, "limit", strconv.Itoa(c.Limit))
	u.RawQuery = vals.Encode()
	us := u.String()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, us, nil)
	if err != nil {
		return nil, err
	}
	return req, nil
}

func (c *GetTagsCommand) ParseResponse(data []byte) (*GetTagsResponse, error) {
	type response struct {
		IsLastPage    bool `json:"isLastPage"`
		Limit         int  `json:"limit"`
		NextPageStart int  `json:"nextPageStart"`
		Size          int  `json:"size"`
		Start         int  `json:"start"`
		Values        []struct {
			ID              string `json:"id"`
			DisplayID       string `json:"displayId"`
			LatestCommit    string `json:"latestCommit"`
			LatestChangeset string `json:"latestChangeset"`
			Type            string `json:"type"`
		} `json:"values"`
	}
	var resp response
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}

	gtr := &GetTagsResponse{
		IsLastPage:    resp.IsLastPage,
		Limit:         resp.Limit,
		NextPageStart: resp.NextPageStart,
		Size:          resp.Size,
		Start:         resp.Start,
	}

	for _, tag := range resp.Values {
		gtr.Tags = append(gtr.Tags, &Tag{
			Name:     tag.DisplayID,
			CommitID: tag.LatestCommit,
			Type:     tag.Type,
		})
	}
	return gtr, nil
}
