package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/myhops/bbfs/nulllog"
)

type SecretString string

func (s SecretString) String() string {
	return strings.Repeat("*", len(s))
}

func (s SecretString) Secret() string {
	return string(s)
}

type Client struct {
	BaseURL   string
	AccessKey SecretString
	Logger    *slog.Logger
}

type GetFileContentCommand struct {
	FilePath   string
	ProjectKey string
	RepoSlug   string
	At         string
}

func (c *Client) initLogger() {
	if c.Logger == nil {
		c.Logger = nulllog.Logger()
	}
}

func (c *GetFileContentCommand) newRequestWithContext(ctx context.Context, baseURL string) (*http.Request, error) {
	u, err := url.Parse(fmt.Sprintf("%s/projects/%s/repos/%s/raw/%s", baseURL, c.ProjectKey, c.RepoSlug, c.FilePath))
	if err != nil {
		return nil, err
	}
	if c.At != "" {
		values := url.Values{}
		values.Add("at", c.At)
		u.RawQuery = values.Encode()
	}
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

func checkStatus(status int) error {
	if status < 200 || status >= 300 {
		return fmt.Errorf("bad status: %s", http.StatusText(status))
	}
	return nil
}

func (c *Client) AuthorizeRequest(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.AccessKey.Secret())
}

func (c *Client) GetFileContent(ctx context.Context, cmd *GetFileContentCommand) ([]byte, error) {
	c.initLogger()
	// Validate the request.
	if err := cmd.Validate(); err != nil {
		return nil, fmt.Errorf("command not valid: %w", err)
	}

	// Build a request.
	req, err := cmd.newRequestWithContext(ctx, c.BaseURL)
	if err != nil {
		return nil, err
	}
	c.AuthorizeRequest(req)

	us := req.URL.String()
	c.Logger.Info("url build", slog.String("url", us))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := checkStatus(resp.StatusCode); err != nil {
		return nil, err
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return cmd.ParseResponse(b)
}

type GetTagsCommand struct {
	ProjectKey string
	RepoSlug   string
	OrderBy    string
}

func (c *GetTagsCommand) newRequestWithContext(ctx context.Context, baseURL string) (*http.Request, error) {
	u, err := url.Parse(fmt.Sprintf("%s/projects/%s/repos/%s/tags", baseURL, c.ProjectKey, c.RepoSlug))
	if err != nil {
		return nil, err
	}
	if c.OrderBy != "" {
		values := url.Values{}
		values.Add("orderBy", c.OrderBy)
		u.RawQuery = values.Encode()
	}
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

func (c *Client) GetTags(ctx context.Context, cmd *GetTagsCommand) ([]string, error) {
	// Validate the request.
	if err := cmd.Validate(); err != nil {
		return nil, fmt.Errorf("command not valid: %w", err)
	}
	// Build a request.
	req, err := cmd.newRequestWithContext(ctx, c.BaseURL)
	if err != nil {
		return nil, err
	}
	c.AuthorizeRequest(req)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := checkStatus(resp.StatusCode); err != nil {
		return nil, err
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return cmd.ParseResponse(b)
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
					Name string `json:"name"`
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

func addValue(v url.Values, name string, value string) {
	if value != "" {
		v.Add(name, value)
	}
}

func (c *GetFilesCommand) newRequestWithContext(ctx context.Context, baseURL string) (*http.Request, error) {
	// https://bitbucket.belastingdienst.nl/rest/api/latest/projects/:projectKey/repos/:repoSlug/browse/:fileName
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

func (c *Client) GetFiles(ctx context.Context, cmd *GetFilesCommand) (*GetFilesResponse, error) {
	// Validate the request.
	if err := cmd.Validate(); err != nil {
		return nil, fmt.Errorf("command not valid: %w", err)
	}
	// Build a request.
	req, err := cmd.newRequestWithContext(ctx, c.BaseURL)
	if err != nil {
		return nil, err
	}
	c.AuthorizeRequest(req)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := checkStatus(resp.StatusCode); err != nil {
		return nil, err
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return cmd.ParseResponse(b)
}

type FilesIterator struct {
	client      *Client
	lastCommand *GetFilesCommand
	lastResult  *GetFilesResponse
	index       int
	lastError   error
	ctx         context.Context
}

func (i *FilesIterator) Next() *FileInfo {
	if i.lastError != nil {
		return nil
	}
	if i.index >= len(i.lastResult.Files) {
		if i.lastResult.LastPage {
			i.lastError = io.EOF
			return nil
		}
		// Get next page.
		if err := i.loadPage(); err != nil {
			i.lastError = err
			return nil
		}
		i.index = 0
	}
	res := i.lastResult.Files[i.index]
	i.index++
	return res
}

func (i *FilesIterator) Err() error {
	return i.lastError
}

func (i *FilesIterator) loadPage() error {
	i.lastCommand.Start = i.lastResult.NextStart
	res, err := i.client.GetFiles(i.ctx, i.lastCommand)
	if err != nil {
		return err
	}
	i.lastResult = res
	return nil
}

func (c *Client) GetFilesIterator(ctx context.Context, cmd *GetFilesCommand) (*FilesIterator, error) {
	// Get the first result and pass it to the iterator.
	res, err := c.GetFiles(ctx, cmd)
	if err != nil {
		return nil, err
	}
	return &FilesIterator{
		client:      c,
		lastResult:  res,
		lastCommand: cmd,
		ctx:         ctx,
	}, nil
}

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

func (c *Client) OpenRawFile(ctx context.Context, cmd *OpenRawFileCommand) (io.ReadCloser, error) {
	// Validate the request.
	if err := cmd.Validate(); err != nil {
		return nil, fmt.Errorf("command not valid: %w", err)
	}
	// Build a request.
	req, err := cmd.newRequestWithContext(ctx, c.BaseURL)
	if err != nil {
		return nil, err
	}
	c.AuthorizeRequest(req)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if err := checkStatus(resp.StatusCode); err != nil {
		resp.Body.Close()
		return nil, err
	}
	return resp.Body, nil
}