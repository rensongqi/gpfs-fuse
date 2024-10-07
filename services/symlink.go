package services

import (
	"bazil.org/fuse"
	"context"
	"os"
	"path/filepath"
)

type Symlink struct {
	fs   *CustomFS
	path string
}

// Attr define soft link file properties
func (s *Symlink) Attr(ctx context.Context, a *fuse.Attr) error {
	fullPath := filepath.Join(s.fs.ExternalStorage, s.path)
	info, err := os.Lstat(fullPath)
	if err != nil {
		return err
	}

	a.Mode = os.ModeSymlink | 0777
	a.Size = uint64(info.Size())
	a.Mtime = info.ModTime()
	return nil
}

// Readlink returns the soft connection target file path
func (s *Symlink) Readlink(ctx context.Context, req *fuse.ReadlinkRequest) (string, error) {
	fullPath := filepath.Join(s.fs.ExternalStorage, s.path)
	linkPath, err := os.Readlink(fullPath)
	if err != nil {
		return "", err
	}
	return linkPath, nil
}
