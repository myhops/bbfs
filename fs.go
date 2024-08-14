package bbfs

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"net/url"
	"path/filepath"
	"time"

	"github.com/myhops/bbfs/bbclient/server"
)

// Define the interface for accessing the Bitbucket repositories.
type Bitbucket interface {
}

const (
	// Default api path for Bitbucket Server
	ApiPath        = "/rest/api"
	// API version
	DefaultVersion = "latest"
)

var (
	ErrNotImplementedYet = errors.New("not implemented yet")
)

// Config contains the configuration for a bitbucket file system.
type Config struct {
	// Host is the hostname of the server
	Host string
	// ProjectKey is the name of the project or the user of the repo
	ProjectKey string
	// RepositorySlug is the name of the repository
	RepositorySlug string
	// Root is the root of the file system in the repo,
	// must be a an existing directory
	Root string
	// AccessKey is an http access key for the repo or the project
	AccessKey string
	// At is a branch, tag or commit
	At string
	// ApiVersion is ignored
	ApiVersion string
}

// Options is the type for NewFS options.
type Option func(*bbFS)

// NewFS returns a new FS.
func NewFS(cfg *Config, opts ...Option) fs.FS {
	version := cfg.ApiVersion
	if version == "" {
		version = DefaultVersion
	}
	u := url.URL{
		Scheme: "https",
		Host:   cfg.Host,
		Path:   filepath.Join(ApiPath, version),
	}

	res := &bbFS{
		client: &server.Client{
			BaseURL:   u.String(),
			AccessKey: server.SecretString(cfg.AccessKey),
		},
		repoSlug:   cfg.RepositorySlug,
		projectKey: cfg.ProjectKey,
		accessKey:  cfg.AccessKey,
		root:       cfg.Root,
		at:         cfg.At,
	}
	for _, o := range opts {
		o(res)
	}
	return res
}

// WithLogger adds a logger to the FS.
func WithLogger(l *slog.Logger) Option {
	return func(f *bbFS) {
		f.client.Logger = l
	}
}

type bbFS struct {
	client     *server.Client
	projectKey string
	repoSlug   string
	accessKey  string
	root       string
	at         string
}

// Sub returns a new FS with dir as root.
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
		at:         b.at,
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
		At:         b.at,
	})
	if err != nil {
		return nil, err
	}

	var found *server.FileInfo
	// Use the new iter over function
	for f := range iter.Files() {
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

// bbFile implements fs.File.
type bbFile struct {
	bfs      *bbFS
	fullPath string
	fi       *bbFileInfo

	data io.ReadCloser

	dirIter *server.FilesIterator
	lastErr error
}

// Read reads from the file.
func (f *bbFile) Read(b []byte) (int, error) {
	if f.data != nil {
		// read the data as a whole
		return f.data.Read(b)
	}

	r, err := f.bfs.client.OpenRawFile(context.Background(), &server.OpenRawFileCommand{
		ProjectKey: f.bfs.projectKey,
		RepoSlug:   f.bfs.repoSlug,
		FilePath:   f.fullPath,
		At:         f.bfs.at,
	})
	if err != nil {
		return 0, err
	}
	f.data = r
	return f.data.Read(b)
}

// Stat returns a FileInfo.
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

// ReadDir returns an array of DirEntry's.
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
			At:         f.bfs.at,
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

// bbfileInfo implements fs.FileInfo and fs.DirEntry
type bbFileInfo struct {
	name    string
	size    int64
	mode    fs.FileMode
	modTime time.Time
}

// Name returns the name of the file.
func (b *bbFileInfo) Name() string {
	return b.name
}

// Name returns the name of the file.
func (b *bbFile) Name() string {
	return b.fi.name
}

// Size returns the size of a regular file or 0 for a directory.
func (b *bbFileInfo) Size() int64 {
	return b.size
}

// Mode returns the Mode of the file.
func (b *bbFileInfo) Mode() fs.FileMode {
	return b.mode
}

// ModTime does not return valuable information.
func (b *bbFileInfo) ModTime() time.Time {
	return b.modTime
}

// IsDir returns true if the file is a directory.
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
