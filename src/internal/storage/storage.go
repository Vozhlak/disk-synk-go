package storage

import (
	"context"
	"io"
)

type File struct {
	Name string
	Type string
}

type Storage interface {
	ListFiles(ctx context.Context) ([]File, error)
	UploadFile(ctx context.Context, file File, content io.Reader) error
}
