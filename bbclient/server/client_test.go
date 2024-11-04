package server

import (
	"context"
	"os"
	"testing"

	"github.com/myhops/bbfs/nulllog"
)

func getAccessKey() string {
	return os.Getenv("BBFSSRV_ACCESS_KEY")
}

func TestGetReadme(t *testing.T) {
	c := &Client{
		BaseURL:   "https://bitbucket.belastingdienst.nl/rest/api/latest",
		AccessKey: SecretString(getAccessKey()),
		Logger: nulllog.Logger(),
	}
	content, err := c.GetFileContent(context.Background(), &GetFileContentCommand{
		ProjectKey: "~zandp06",
		RepoSlug:   "testraw",
		FilePath:   "README.md",
		At:         "olo-kor-eb-service/1.0.0.1",
	})
	if err != nil {
		t.Fatalf("error: %s", err.Error())
	}
	t.Logf("%s", string(content))
}

func TestGetTags(t *testing.T) {
	c := &Client{
		BaseURL:   "https://bitbucket.belastingdienst.nl/rest/api/latest",
		AccessKey: SecretString(getAccessKey()),
	}
	tags, err := c.GetTags(context.Background(), &GetTagsCommand{
		ProjectKey: "~zandp06",
		RepoSlug:   "testraw",
		OrderBy:    "ALPHABETICAL",
	})
	if err != nil {
		t.Fatalf("error: %s", err.Error())
	}
	t.Logf("%v", tags)
}

func TestGetFiles(t *testing.T) {
	c := &Client{
		BaseURL:   "https://bitbucket.belastingdienst.nl/rest/api/latest",
		AccessKey: SecretString(getAccessKey()),
	}
	files, err := c.GetFiles(context.Background(), &GetFilesCommand{
		ProjectKey: "~zandp06",
		RepoSlug:   "testraw",
		FilePath: "",
		Limit: 100,
	})
	if err != nil {
		t.Fatalf("error: %s", err.Error())
	}
	t.Logf("%v", files)
}

func TestGetAllFiles(t *testing.T) {
	c := &Client{
		BaseURL:   "https://bitbucket.belastingdienst.nl/rest/api/latest",
		AccessKey: SecretString(getAccessKey()),
	}
	files, err := c.GetFiles(context.Background(), &GetFilesCommand{
		ProjectKey: "~zandp06",
		RepoSlug:   "testraw",
		FilePath: "",
		Limit: 100,
	})
	if err != nil {
		t.Fatalf("error: %s", err.Error())
	}
	t.Logf("%v", files)
}

func TestIterator(t *testing.T) {
	c := &Client{
		BaseURL:   "https://bitbucket.belastingdienst.nl/rest/api/latest",
		AccessKey: SecretString(getAccessKey()),
	}
	iter, err := c.GetFilesIterator(context.Background(), &GetFilesCommand{
		ProjectKey: "~zandp06",
		RepoSlug:   "testraw",
		FilePath: "server",
		Limit: 7,
	})
	if err != nil {
		t.Fatalf("error: %s", err.Error())
	}
	var n int
	for f := iter.Next(); f != nil; f = iter.Next() {
		n++
		t.Logf("%v", f)
	}
	t.Logf("%d", n)
}


