package server

import (
	"context"
	"io"
	"iter"
)

// FilesIterator is an iterator for the files in a directory in the repository.
type FilesIterator struct {
	client      *Client
	lastCommand *GetFilesCommand
	lastResult  *GetFilesResponse
	index       int
	lastError   error
	ctx         context.Context
}

// Next returns the next FileInfo in the directory, or nil if all entries have been read.
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

// Err returns the last occured error.
func (i *FilesIterator) Err() error {
	return i.lastError
}

// loadPage loads the next page from the directory.
func (i *FilesIterator) loadPage() error {
	i.lastCommand.Start = i.lastResult.NextStart
	res, err := i.client.GetFiles(i.ctx, i.lastCommand)
	if err != nil {
		return err
	}
	i.lastResult = res
	return nil
}

// Files returns a new iter iterator
func (i *FilesIterator) Files() iter.Seq[*FileInfo] {
	return func(yield func(v *FileInfo) bool) {
		for f := i.Next(); f != nil; f = i.Next() {
			if !yield(f) {
				return
			}
		}
	}
}