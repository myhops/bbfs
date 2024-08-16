package server

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/myhops/bbfs/nulllog"
)

// Secret string masks the value for String to avoid accidental disclosure.
type SecretString string

// String returns a masked value.
func (s SecretString) String() string {
	return strings.Repeat("*", len(s))
}

// Secret returns the secret value.
func (s SecretString) Secret() string {
	return string(s)
}

// Client is a client for the Bitbucket repository.
type Client struct {
	BaseURL   string
	AccessKey SecretString
	Logger    *slog.Logger
}

func (c *Client) initLogger() {
	if c.Logger == nil {
		c.Logger = nulllog.Logger()
	}
}

func checkStatus(status int) error {
	if status < 200 || status >= 300 {
		return fmt.Errorf("bad status: %s", http.StatusText(status))
	}
	return nil
}

// AuthorizeRequest adds an Authorization bearer header to the headers.
func (c *Client) AuthorizeRequest(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.AccessKey.Secret())
}

// GetFileContent retrieves text content from the file.
//
// Use OpenRawFile if you want to read the file content.
func (c *Client) GetFileContent(ctx context.Context, cmd *GetFileContentCommand) ([]byte, error) {
	c.initLogger()
	return DoCommandResponse[*GetFileContentCommand, []byte](ctx, c, cmd)
}

// GetTags returns the tags in the repository.
func (c *Client) GetTags(ctx context.Context, cmd *GetTagsCommand) (*GetTagsResponse, error) {
	return DoCommandResponse(ctx, c, cmd)
}

// GetCommits returns an array of commits or a single commit.
func (c *Client) GetCommits(ctx context.Context, cmd *GetCommitsCommand) (*GetCommitsResponse, error) {
	return DoCommandResponse(ctx, c, cmd)
}

func addValue(v url.Values, name string, value string) {
	if value != "" {
		v.Add(name, value)
	}
}

// GetFiles returns a GetFilesResponse that contains the list of files found.
func (c *Client) GetFiles(ctx context.Context, cmd *GetFilesCommand) (*GetFilesResponse, error) {
	return DoCommandResponse(ctx, c, cmd)
}

// GetFilesIterator returns a file interator for the FilePath in GetFilesCommand.
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

type command interface {
	Validate() error
	newRequestWithContext(ctx context.Context, baseURL string) (*http.Request, error)
}

type commandResponse[T any] interface {
	command
	ParseResponse([]byte) (T, error)
}

// DoCommandBody performs Do for the given command and returns the response body.
// You need to close the io.ReadCloser after use.
func DoCommandBody(ctx context.Context, client *Client, cmd command) (io.ReadCloser, error) {
	// Validate the request.
	if err := cmd.Validate(); err != nil {
		return nil, fmt.Errorf("command not valid: %w", err)
	}
	// Build a request.
	req, err := cmd.newRequestWithContext(ctx, client.BaseURL)
	if err != nil {
		return nil, err
	}
	client.AuthorizeRequest(req)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if err := checkStatus(resp.StatusCode); err != nil {
		return nil, err
	}
	return resp.Body, nil
}

// DoCommandResponse performs do for the given command and returns the parsed body.
func DoCommandResponse[C commandResponse[T], T any](ctx context.Context, client *Client, cmd C) (T, error) {
	var nullRes T
	body, err := DoCommandBody(ctx, client, cmd)
	if err != nil {
		return nullRes, err
	}
	defer body.Close()

	b, err := io.ReadAll(body)
	if err != nil {
		return nullRes, err
	}
	return cmd.ParseResponse(b)
}

// OpenRawFile opens the file as specified in the cmd parameter. 
// The returned io.ReadCloser is the body of the response.
// You need to close the io.ReadCloser after use.
func (c *Client) OpenRawFile(ctx context.Context, cmd *OpenRawFileCommand) (io.ReadCloser, error) {
	return DoCommandBody(ctx, c, cmd)
}
