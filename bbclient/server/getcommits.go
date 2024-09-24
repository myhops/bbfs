package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type Committer struct {
	Name  string
	EMail string
}

type Commit struct {
	Committer Committer
	Timestamp time.Time
	Message   string
}

type GetCommitsCommand struct {
	ProjectKey string
	RepoSlug   string
	OrderBy    string
	Start      int
	Limit      int
	CommitID   string // optional
}

type GetCommitsResponse struct {
	Commits []*Commit
}

func (c *GetCommitsCommand) Validate() error {
	if c.ProjectKey == "" {
		return fmt.Errorf("ProjectKey is missing")
	}
	if c.RepoSlug == "" {
		return fmt.Errorf("RepoSlug is missing")
	}
	return nil
}

func (c *GetCommitsCommand) newRequestWithContext(ctx context.Context, baseURL string) (*http.Request, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("error parsing base url for GetCommitsCommand: %w", err)
	}
	u = u.JoinPath("projects", c.ProjectKey, "repos", c.RepoSlug, "commits", c.CommitID)

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

func (c *GetCommitsCommand) ParseResponse(data []byte) (*GetCommitsResponse, error) {
	type Actor struct {
		Name         string `json:"name"`
		EmailAddress string `json:"emailAddress"`
	}
	type Value struct {
		ID                 string    `json:"id"`
		Author             Actor     `json:"author"`
		AuthorTimestamp    time.Time `json:"authorTimestamp"`
		Committer          Actor     `json:"committer"`
		CommitterTimestamp time.Time `json:"committerTimestamp"`
		Message            string    `json:"message"`
	}
	type Response struct {
		Size          int     `json:"size"`
		IsLastPage    bool    `json:"isLastPage"`
		NextPageStart int     `json:"nextPageStart"`
		Start         int     `json:"start"`
		Values        []Value `json:"values"`
	}

	parseValue := func(data []byte) (*Value, error) {
		var res Value
		if err := json.Unmarshal(data, &res); err != nil {
			return nil, fmt.Errorf("error unmarshalling single commit: %w", err)
		}
		return &res, nil
	}

	parseResponse := func(data []byte) (*Response, error) {
		var res Response
		if err := json.Unmarshal(data, &res); err != nil {
			return nil, fmt.Errorf("error unmarshalling list of commits: %w", err)
		}
		return &res, nil
	}

	// Check if the response is for a list of commits
	if c.CommitID == "" {
		v, err := parseValue(data)
		if err != nil {
			return nil, err
		}
		return &GetCommitsResponse{
			Commits: []*Commit{
				{
					Committer: Committer{
						Name:  v.Committer.Name,
						EMail: v.Committer.EmailAddress,
					},
					Timestamp: v.CommitterTimestamp,
					Message:   v.Message,
				},
			},
		}, nil
	}

	// Parse a list of commits
	resp, err := parseResponse(data)
	if err != nil {
		return nil, err
	}

	var commits []*Commit
	for _, v := range resp.Values {
		commits = append(commits, 
			&Commit{
					Committer: Committer{
						Name:  v.Committer.Name,
						EMail: v.Committer.EmailAddress,
					},
					Timestamp: v.CommitterTimestamp,
					Message:   v.Message,
				},
			)
	}
	return &GetCommitsResponse{
		Commits: commits,
	}, nil;
}

