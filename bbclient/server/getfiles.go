package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

type GetFilesCommand struct {
	FilePath   string
	ProjectKey string
	RepoSlug   string
	At         string
	Start      int
	Limit      int
}

type GetFilesResponse struct {
	Files     []*FileInfo
	Start     int
	NextStart int
	LastPage  bool
	Size      int
}

func (c *GetFilesCommand) Validate() error {
	if c.ProjectKey == "" {
		return fmt.Errorf("ProjectKey is missing")
	}
	if c.RepoSlug == "" {
		return fmt.Errorf("RepoSlug is missing")
	}
	return nil
}

func (c *GetFilesCommand) ParseResponse(data []byte) (*GetFilesResponse, error) {
	var r struct {
		Children struct {
			Size          int  `json:"size"`
			IsLastPage    bool `json:"isLastPage"`
			NextPageStart int  `json:"nextPageStart"`
			Start         int  `json:"start"`
			Values        []struct {
				Path struct {
					Name       string   `json:"name"`
					Components []string `json:"components"`
				} `json:"path"`
				Type string `json:"type"`
				Size int64  `json:"size"`
			} `json:"values"`
		} `json:"children"`
	}
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, err
	}
	resp := &GetFilesResponse{
		Start:     r.Children.Start,
		Size:      r.Children.Size,
		NextStart: r.Children.NextPageStart,
		LastPage:  r.Children.IsLastPage,
	}
	for _, v := range r.Children.Values {
		resp.Files = append(resp.Files, &FileInfo{
			Name: v.Path.Components[0],
			Size: v.Size,
			Type: v.Type,
		})
	}
	return resp, nil
}

func (c *GetFilesCommand) newRequestWithContext(ctx context.Context, baseURL string) (*http.Request, error) {
	u, err := url.Parse(fmt.Sprintf("%s/projects/%s/repos/%s/browse/%s", baseURL, c.ProjectKey, c.RepoSlug, c.FilePath))
	if err != nil {
		return nil, err
	}
	vals := url.Values{}
	addValue(vals, "at", c.At)
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

type FileInfo struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
	Type string `json:"type"`
}
