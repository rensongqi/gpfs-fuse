package services

import (
	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"context"
	"github.com/minio/minio-go/v7"
	logger "github.com/sirupsen/logrus"
	"gpfs-fuse/settings"
	"os"
	"path/filepath"
	"syscall"
)

type Dir struct {
	fs   *CustomFS // custom vfs
	ln   *Symlink  // link file
	path string
}

// Attr of dir
func (d *Dir) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Mode = os.ModeDir | 0755
	return nil
}

// Lookup 
// Iterate over directories to get file properties
// The ls -l command displays the metadata information
func (d *Dir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	path := filepath.Join(d.path, name)
	fullPath := filepath.Join(d.fs.ExternalStorage, path)
	fi, err := os.Lstat(fullPath)
	if err != nil {
		return nil, syscall.ENOENT
	}

	if fi.Mode()&os.ModeSymlink != 0 {
		if hasSpecialAttribute(fullPath) {
			return &File{fs: d.fs, path: path, isSymlink: true}, nil
		}
		return &Symlink{fs: d.fs, path: path}, nil
	}

	if fi.IsDir() {
		return &Dir{fs: d.fs, path: path}, nil
	}
	return &File{fs: d.fs, path: path}, nil
}

// Create file
func (d *Dir) Create(ctx context.Context, req *fuse.CreateRequest, resp *fuse.CreateResponse) (fs.Node, fs.Handle, error) {
	path := filepath.Join(d.path, req.Name)
	fullPath := filepath.Join(d.fs.ExternalStorage, path)
	_, err := os.Create(fullPath)
	if err != nil {
		return nil, nil, err
	}
	node := &File{fs: d.fs, path: path}
	return node, node, nil
}

// Remove file
func (d *Dir) Remove(ctx context.Context, req *fuse.RemoveRequest) error {
	path := filepath.Join(d.path, req.Name)
	fullPath := filepath.Join(d.fs.ExternalStorage, path)
	err := os.RemoveAll(fullPath)
	if err != nil {
		return err
	}
	// If the file exists in oss, it also needs to be deleted
	if hasSpecialAttribute(path) {
		err = d.fs.MinioClient.RemoveObject(ctx, settings.Bucket, path, minio.RemoveObjectOptions{GovernanceBypass: true, ForceDelete: true})
		if err != nil {
			logger.Error("delete oss file err: ", err)
			return err
		}
	}
	return nil
}

// Mkdir
func (d *Dir) Mkdir(ctx context.Context, req *fuse.MkdirRequest) (fs.Node, error) {
	path := filepath.Join(d.path, req.Name)
	fullPath := filepath.Join(d.fs.ExternalStorage, path)

	err := os.Mkdir(fullPath, req.Mode)
	if err != nil {
		return nil, err
	}

	return &Dir{fs: d.fs, path: path}, nil
}

// ReadDirAll 
func (d *Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	fullPath := filepath.Join(d.fs.ExternalStorage, d.path)
	files, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, err
	}
	var dirents []fuse.Dirent
	for _, file := range files {
		var dirent fuse.Dirent
		dirent.Name = file.Name()
		if file.IsDir() {
			dirent.Type = fuse.DT_Dir
		} else {
			dirent.Type = fuse.DT_File
		}
		dirents = append(dirents, dirent)
	}
	return dirents, nil
}
