package localfs

import (
	"context"
	"disk-sync/src/internal/storage"
	"fmt"
	"io"
	"os"
)

type Client struct {
	rootDir string
}

func New(dir string) *Client {
	return &Client{
		rootDir: dir,
	}
}

func (c *Client) ListFiles(ctx context.Context) ([]storage.File, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("list local files: %w", err)
	}

	entries, err := os.ReadDir(c.rootDir)
	if err != nil {
		return nil, fmt.Errorf("list local files in %q: %w", c.rootDir, err)
	}

	if err = ctx.Err(); err != nil {
		return nil, fmt.Errorf("list local files: %w", err)
	}

	files := make([]storage.File, 0, len(entries))

	for _, entry := range entries {
		fileType := "file"
		if entry.IsDir() {
			fileType = "dir"
		}

		files = append(files, storage.File{
			Name: entry.Name(),
			Type: fileType,
		})
	}

	return files, nil
}

func (c *Client) UploadFile(ctx context.Context, file storage.File, content io.Reader) error {
	//TODO implement me
	panic("implement me")
}
