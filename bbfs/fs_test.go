package bbfs

import (
	"io/fs"
	"testing"
	"testing/fstest"
)

const (
	accessKey = ""
)

var testCfg = &Config{
	Host: "bitbucket.org",
	ProjectKey: "myhops",
	RepositorySlug: "testflags",
	AccessKey: accessKey,
}

func TestMe(t *testing.T) {
	bfs := NewFS(testCfg)
	if err := fstest.TestFS(bfs, "hugo/TestBitbucketAccess", "hugo", "go.mod"); err != nil {
		t.Errorf("%s", err.Error())
	}
}

func TestOpen(t *testing.T) {
	bfs := NewFS(testCfg)
	{
		f, err := bfs.Open("index.html")
		if err != nil {
			t.Fatalf("%s", err.Error())
		}
		f.Close()
	}

	{
		f, err := bfs.Open("server")
		if err != nil {
			t.Fatalf("%s", err.Error())
		}
		f.Close()
	}
}

func TestGlob(t *testing.T) {
	bfs := NewFS(testCfg)
	matches, err := fs.Glob(bfs, "*hug*")
	if err != nil {
		t.Fatalf("%s", err.Error())
	}
	t.Logf("%#v", matches)
	// t.Error()
}
