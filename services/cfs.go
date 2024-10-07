package services

import (
	"bazil.org/fuse/fs"
	"context"
	"github.com/minio/minio-go/v7"
	"gpfs-fuse/settings"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
)

type CustomFS struct {
	ExternalStorage string
	MinioClient     *minio.Client
	FileHandles     sync.Map
}

// Root defines root
func (cfs *CustomFS) Root() (fs.Node, error) {
	return &Dir{fs: cfs, path: ""}, nil
}

// Read file from minio
func (cfs *CustomFS) readFromMinio(path string) ([]byte, error) {
	obj, err := cfs.MinioClient.GetObject(context.Background(), settings.Bucket, path, minio.GetObjectOptions{})
	if err != nil {
		log.Println("read from minio err: ", err)
		return nil, err
	}
	defer func() {
		_ = obj.Close()
	}()
	data, err := io.ReadAll(obj)
	if err != nil {
		return nil, err
	}
	return data, nil
}
