package bbfs

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"net/url"
	"path/filepath"
	"time"

	"github.com/myhops/bbfs/bbclient/server"
)

// Define the interface for accessing the Bitbucket repositories.
type Bitbucket interface {
}

const (
	ApiPath        = "/rest/api"
	DefaultVersion = "latest"
)

var (
	ErrNotImplementedYet = errors.New("not implemented yet")
)

type Config struct {
	Host           string
	ProjectKey     string
	RepositorySlug string
	Root           string
	AccessKey      string
	At             string
	ApiVersion     string
}

func NewFS(cfg *Config) *bbFS {
	version := cfg.ApiVersion
	if version == "" {
		version = DefaultVersion
	}
	u := url.URL{
		Scheme: "https",
		Host:   cfg.Host,
		Path:   filepath.Join(ApiPath, version),
	}

	return &bbFS{
		client: &server.Client{
			BaseURL:   u.String(),
			AccessKey: server.SecretString(cfg.AccessKey),
		},
		repoSlug:   cfg.RepositorySlug,
		projectKey: cfg.ProjectKey,
		accessKey:  cfg.AccessKey,
		root:       cfg.Root,
	}
}

type bbFS struct {
	client     *server.Client
	projectKey string
	repoSlug   string
	accessKey  string
	root       string
}

func (b *bbFS) Sub(dir string) (fs.FS, error) {
	// check if the dir exists.
	f, err := b.Open(dir)
	if err != nil {
		return nil, err
	}

	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !fi.IsDir() {
		return nil, fs.ErrInvalid
	}

	return &bbFS{
		root:       filepath.Join(b.root, fi.Name()),
		client:     b.client,
		projectKey: b.projectKey,
		repoSlug:   b.repoSlug,
		accessKey:  b.accessKey,
	}, nil
}

func isModeDir(t string) fs.FileMode {
	if t == "DIRECTORY" {
		return fs.ModeDir
	}
	return 0
}

// Open opens the file on the repository.
func (b *bbFS) Open(name string) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{
			Path: name,
			Op:   "open",
			Err:  fs.ErrInvalid,
		}
	}

	// Get the directory listing of the parent path.
	fullPath := filepath.Join(b.root, name)
	parent := filepath.Dir(fullPath)
	base := filepath.Base(fullPath)

	// Test if in root.
	if fullPath == "." {
		return &bbFile{
			fullPath: fullPath,
			bfs:      b,
			fi: &bbFileInfo{
				name: ".",
				mode: fs.ModeDir,
			},
		}, nil
	}
	if parent == "." {
		parent = ""
	}

	// Check if the file exists in the directory.
	iter, err := b.client.GetFilesIterator(context.Background(), &server.GetFilesCommand{
		FilePath:   parent,
		ProjectKey: b.projectKey,
		RepoSlug:   b.repoSlug,
		Limit:      1000,
	})
	if err != nil {
		return nil, err
	}

	var found *server.FileInfo
	for f := iter.Next(); f != nil; f = iter.Next() {
		if f.Name == base {
			found = f
		}
	}
	if found == nil {
		return nil, fs.ErrNotExist
	}

	// Create the file.
	res := &bbFile{
		fullPath: fullPath,
		bfs:      b,
		fi: &bbFileInfo{
			name: found.Name,
			mode: isModeDir(found.Type),
			size: found.Size,
		},
	}
	if res.IsDir() {
		res.fi.mode = fs.ModeDir
	}
	return res, nil
}

type bbFile struct {
	bfs      *bbFS
	fullPath string
	fi       *bbFileInfo

	data io.ReadCloser

	dirIter *server.FilesIterator
	lastErr error
}

func (f *bbFile) Read(b []byte) (int, error) {
	if f.data != nil {
		// read the data as a whole
		return f.data.Read(b)
	}

	r, err := f.bfs.client.OpenRawFile(context.Background(), &server.OpenRawFileCommand{
		ProjectKey: f.bfs.projectKey,
		RepoSlug:   f.bfs.repoSlug,
		FilePath:   f.fullPath,
	})
	if err != nil {
		return 0, err
	}
	f.data = r
	return f.data.Read(b)
}

func (f *bbFile) Stat() (fs.FileInfo, error) {
	return f.fi, nil
}

func (f *bbFile) Close() error {
	if f.data == nil {
		return nil
	}
	tmp := f.data
	f.data = nil
	return tmp.Close()
}

func (f *bbFile) ReadDir(n int) ([]fs.DirEntry, error) {
	if f.lastErr != nil {
		return nil, f.lastErr
	}
	fullPath := f.fullPath
	if fullPath == "." {
		fullPath = ""
	}
	if f.dirIter == nil {
		iter, err := f.bfs.client.GetFilesIterator(context.Background(), &server.GetFilesCommand{
			FilePath:   fullPath,
			ProjectKey: f.bfs.projectKey,
			RepoSlug:   f.bfs.repoSlug,
			Limit:      1000,
		})
		if err != nil {
			return nil, err
		}
		f.dirIter = iter
	}

	res := []fs.DirEntry{}
	var i int
	for {
		// Check if done
		if n > 0 && i == n {
			return res, nil
		}

		ff := f.dirIter.Next()
		if ff == nil {
			if !errors.Is(f.dirIter.Err(), io.EOF) {
				f.lastErr = f.dirIter.Err()
				return res, f.dirIter.Err()
			}
			if len(res) == 0 {
				f.lastErr = io.EOF
			}
			return res, nil
		}

		bf := &bbFile{
			fullPath: filepath.Join(f.fullPath, ff.Name),
			fi: &bbFileInfo{
				name: ff.Name,
				mode: isModeDir(ff.Type),
				size: ff.Size,
			},
		}
		if bf.IsDir() {
			bf.fi.mode = fs.ModeDir
		}
		res = append(res, bf)
		i++
	}
}

type bbFileInfo struct {
	name    string
	size    int64
	mode    fs.FileMode
	modTime time.Time
}

func (b *bbFileInfo) Name() string {
	return b.name
}

func (b *bbFile) Name() string {
	return b.fi.name
}

func (b *bbFileInfo) Size() int64 {
	return b.size
}

func (b *bbFileInfo) Mode() fs.FileMode {
	return b.mode
}

func (b *bbFileInfo) ModTime() time.Time {
	return b.modTime
}

func (b *bbFileInfo) IsDir() bool {
	return b.mode.IsDir()
}

func (b *bbFile) IsDir() bool {
	return b.fi.IsDir()
}

func (b *bbFileInfo) Sys() any {
	return nil
}

func (f *bbFile) Type() fs.FileMode {
	return f.fi.mode
}

func (f *bbFile) Info() (fs.FileInfo, error) {
	return f.fi, nil
}

var _ fs.DirEntry = &bbFile{}
var _ fs.ReadDirFile = &bbFile{}
