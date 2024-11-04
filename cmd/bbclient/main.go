package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/myhops/bbfs/bbclient/server"
	"github.com/myhops/bbfs/nulllog"
)

// options contains the options for all commands
type options struct {
	Command    string
	BaseURL    string
	AccessKey  server.SecretString
	ProjectKey string
	RepoSlug   string
	OrderBy    string
	Limit      int
	FilePath   string
	At         string
	CommitID   string
}

func defaultOptions() *options {
	return &options{
		BaseURL: "https://bitbucket.belastingdienst.nl/rest/api/latest",
		OrderBy: "MODIFICATION",
	}
}

// setIfSet sets the string if v is not empty
func setIfSet(v string, val *string) {
	if v != "" {
		*val = v
	}
}

// setIfSetInt sets val if v is not empty and an int value
func setIfSetInt(v string, val *int) {
	if v != "" {
		i, err := strconv.Atoi(v)
		if err != nil {
			return
		}
		*val = i
	}
}

func setIfSetSecretString(v string, val *server.SecretString) {
	if v != "" {
		*val = server.SecretString(v)
	}
}

func setFromEnv(opts *options, getenv func(string) string) {
	setIfSet(getenv("BBFS_CLIENT_COMMAND"), &opts.Command)
	setIfSet(getenv("BBFS_CLIENT_BASE_URL"), &opts.BaseURL)
	setIfSetSecretString(getenv("BBFS_CLIENT_ACCESS_KEY"), &opts.AccessKey)
	setIfSet(getenv("BBFS_CLIENT_PROJECT_KEY"), &opts.ProjectKey)
	setIfSet(getenv("BBFS_CLIENT_REPO_SLUG"), &opts.RepoSlug)
	setIfSet(getenv("BBFS_CLIENT_ORDER_BY"), &opts.OrderBy)
	setIfSetInt(getenv("BBFS_CLIENT_LIMIT"), &opts.Limit)
	setIfSet(getenv("BBFS_CLIENT_FILE_PATH"), &opts.FilePath)
	setIfSet(getenv("BBFS_CLIENT_AT"), &opts.At)
	setIfSet(getenv("BBFS_CLIENT_COMMIT_ID"), &opts.CommitID)
}

func setFromArgs(opts *options, args []string) error {
	// Get the flags
	fs := flag.NewFlagSet("temp", flag.ContinueOnError)
	command := fs.String("command", "", "The command to execute")
	baseURL := fs.String("base-url", "", "Base url of the bitbucket server on premises,\ndefaults to https://bitbucket.belastingdienst.nl/rest/api/latest")
	accessKey := fs.String("access-key", "", "Access key for the repository")
	projectKey := fs.String("project-key", "", "The bitbucket project or the user name")
	repoSlug := fs.String("repo-slug", "", "repo name")
	orderBy := fs.String("order-by", "", "Order by [ ALPHABETICAL | ... ]")
	limit := fs.String("limit", "", "Maximum number of entries to return, defauls to 25")
	filePath := fs.String("file-path", "", "File path")
	at := fs.String("at", "", "branch or tag")
	commitID := fs.String("commit-id", "", "commit id")

	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	getenv := func(key string) string {
		switch key {
		case "BBFS_CLIENT_COMMAND":
			return *command
		case "BBFS_CLIENT_BASE_URL":
			return *baseURL
		case "BBFS_CLIENT_PROJECT_KEY":
			return *projectKey
		case "BBFS_CLIENT_REPO_SLUG":
			return *repoSlug
		case "BBFS_CLIENT_ORDER_BY":
			return *orderBy
		case "BBFS_CLIENT_FILE_PATH":
			return *filePath
		case "BBFS_CLIENT_AT":
			return *at
		case "BBFS_CLIENT_COMMIT_ID":
			return *commitID
		case "BBFS_CLIENT_ACCESS_KEY":
			return *accessKey
		case "BBFS_CLIENT_LIMIT":
			return *limit
		}
		return ""
	}

	setFromEnv(opts, getenv)

	return nil
}

func getClient(opts *options) *server.Client {
	c := &server.Client{
		BaseURL:   opts.BaseURL,
		AccessKey: opts.AccessKey,
		Logger:    nulllog.Logger(),
	}
	return c
}

func cmdGetTags(opts *options) error {
	// Create client
	client := getClient(opts)

	// Create the command
	cmd := &server.GetTagsCommand{
		ProjectKey: opts.ProjectKey,
		RepoSlug:   opts.RepoSlug,
		Limit:      opts.Limit,
		OrderBy:    opts.OrderBy,
	}

	// execute command
	resp, err := client.GetTags(context.Background(), cmd)
	if err != nil {
		return err
	}

	// Print the result.
	for _, e := range resp.Tags {
		fmt.Printf("name %s, type %s\n", e.Name, e.Type)
	}
	return nil
}

func run(args []string, getenv func(string) string) error {
	opts := defaultOptions()
	setFromEnv(opts, getenv)
	if err := setFromArgs(opts, args); err != nil {
		return err
	}

	if opts.Command == "" {
		return fmt.Errorf("no command specified")
	}

	switch cmd := opts.Command; cmd {
	case "tags":
		return cmdGetTags(opts)
	}

	return fmt.Errorf("bad command: %s", opts.Command)
}

func main() {

	if err := run(os.Args, os.Getenv); err != nil {
		log.Fatalf("run failed: %s", err.Error())
	}

}
