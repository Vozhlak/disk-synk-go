package storage

import "context"

type File struct {
	Name string
	Type string
}

type Storage interface {
	ListFiles(ctx context.Context) ([]File, error)
}
