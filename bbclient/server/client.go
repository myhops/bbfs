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

func (c *Client) AuthorizeRequest(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.AccessKey.Secret())
}

func (c *Client) GetFileContent(ctx context.Context, cmd *GetFileContentCommand) ([]byte, error) {
	c.initLogger()
	return DoCommandResponse[*GetFileContentCommand, []byte](ctx, c, cmd)
}

func (c *Client) GetTags(ctx context.Context, cmd *GetTagsCommand) ([]string, error) {
	return DoCommandResponse(ctx, c, cmd)
}

func addValue(v url.Values, name string, value string) {
	if value != "" {
		v.Add(name, value)
	}
}

func (c *Client) GetFiles(ctx context.Context, cmd *GetFilesCommand) (*GetFilesResponse, error) {
	return DoCommandResponse(ctx, c, cmd)
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

type command interface {
	Validate() error
	newRequestWithContext(ctx context.Context, baseURL string) (*http.Request, error)
}

type commandResponse[T any] interface {
	command
	ParseResponse([]byte) (T, error)
}

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

func (c *Client) OpenRawFile(ctx context.Context, cmd *OpenRawFileCommand) (io.ReadCloser, error) {
	return DoCommandBody(ctx, c, cmd)
}
