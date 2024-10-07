package services

import (
	"bazil.org/fuse"
	"context"
	"fmt"
	"github.com/minio/minio-go/v7"
	logger "github.com/sirupsen/logrus"
	"gpfs-fuse/settings"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

type File struct {
	fs         *CustomFS
	mu         sync.Mutex
	path       string
	isSymlink  bool
	cachedData []byte
}

// Attr defines common file properties
func (f *File) Attr(ctx context.Context, a *fuse.Attr) error {
	fullPath := filepath.Join(f.fs.ExternalStorage, f.path)
	if f.isSymlink {
		info, err := os.Lstat(fullPath)
		if err != nil {
			return err
		}

		a.Mode = os.FileMode(448)
		a.Size = getLinkFileSize(fullPath)
		a.Mtime = info.ModTime()
	} else {
		fi, err := os.Stat(fullPath)
		if err != nil {
			return err
		}
		a.Size = uint64(fi.Size())
		a.Mode = fi.Mode()
		a.Mtime = fi.ModTime()
	}
	return nil
}

// Read file
func (f *File) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.isSymlink {
		if f.cachedData == nil {
			fullPath := filepath.Join(f.fs.ExternalStorage, f.path)
			linkPath, err := os.Readlink(fullPath)
			if err != nil {
				logger.Error("file read lstat err: ", err)
				return err
			}

			var object string
			linkPathSplit := strings.Split(linkPath, "/")
			l := len(linkPathSplit) - 1
			for i := 3; i <= l; i++ {
				if i != l {
					object += linkPathSplit[i] + "/"
				} else {
					object += linkPathSplit[i]
				}
			}

			if err = f.cacheFileFromMinIO(ctx, object); err != nil {
				return err
			}
		}

		return f.readFromCache(req, resp)
	}

	handle, ok := f.fs.FileHandles.Load(f.path)
	if !ok {
		fullPath := filepath.Join(f.fs.ExternalStorage, f.path)
		file, err := os.Open(fullPath)
		if err != nil {
			return err
		}
		f.fs.FileHandles.Store(f.path, file)
		handle = file
	}

	file := handle.(*os.File)
	_, err := file.Seek(req.Offset, io.SeekStart)
	if err != nil {
		return err
	}

	resp.Data = make([]byte, req.Size)
	n, err := file.Read(resp.Data)
	if err != nil && err != io.EOF {
		return err
	}
	resp.Data = resp.Data[:n]
	return nil
}

// Read cached data from the minio
func (f *File) cacheFileFromMinIO(ctx context.Context, object string) error {
	obj, err := f.fs.MinioClient.GetObject(ctx, settings.Bucket, object, minio.GetObjectOptions{})
	if err != nil {
		logger.Errorf("failed to get object %s from minio: %v", object, err)
		return fmt.Errorf("failed to get object from MinIO: %v", err)
	}
	defer func() {
		_ = obj.Close()
	}()
	// Read the entire file into the cache
	f.cachedData, err = io.ReadAll(obj)
	if err != nil {
		return err
	}
	return nil
}

func (f *File) readFromCache(req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	if req.Offset >= int64(len(f.cachedData)) {
		resp.Data = []byte{}
		return nil
	}

	end := req.Offset + int64(req.Size)
	if end > int64(len(f.cachedData)) {
		end = int64(len(f.cachedData))
	}

	resp.Data = make([]byte, end-req.Offset)
	copy(resp.Data, f.cachedData[req.Offset:end])

	go func() {
		time.Sleep(time.Second * 60)
		f.cleanCache()
	}()

	return nil
}

func (f *File) cleanCache() {
	f.cachedData = nil
}

func (f *File) Release(ctx context.Context, req *fuse.ReleaseRequest) error {
	if handle, ok := f.fs.FileHandles.Load(f.path); ok {
		_ = handle.(*os.File)
		f.fs.FileHandles.Delete(f.path)
		return nil
	}
	return nil
}

// Write file
func (f *File) Write(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
	fullPath := filepath.Join(f.fs.ExternalStorage, f.path)
	file, err := os.OpenFile(fullPath, os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	n, err := file.WriteAt(req.Data, req.Offset)
	if err != nil {
		return err
	}
	resp.Size = n
	return nil
}

// Setattr Implement the function of modifying file attributes, including chmod
func (f *File) Setattr(ctx context.Context, req *fuse.SetattrRequest, resp *fuse.SetattrResponse) error {
	fullPath := filepath.Join(f.fs.ExternalStorage, f.path)

	if req.Valid.Mode() {
		if err := os.Chmod(fullPath, req.Mode); err != nil {
			return err
		}
	}
	if req.Valid.Size() {
		if err := syscall.Truncate(fullPath, int64(req.Size)); err != nil {
			return err
		}
	}

	return nil
}
