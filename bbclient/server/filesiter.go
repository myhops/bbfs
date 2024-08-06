package server

import (
	"context"
	"io"
)

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
