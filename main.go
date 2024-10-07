package main

import (
	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"flag"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	logger "github.com/sirupsen/logrus"
	_ "gpfs-fuse/pkg"
	"gpfs-fuse/services"
	"gpfs-fuse/settings"
	"os"
	"path/filepath"
)

func main() {
	externalStorage := flag.String("external", "", "Path to external storage")
	mountPoint := flag.String("mount", "", "Mount point for FUSE filesystem")

	flag.Parse()
	if *externalStorage == "" || *mountPoint == "" {
		logger.Fatal("External storage path and mount point must be provided")
	}

	// 创建挂载点
	_, err := os.Stat(*mountPoint)
	if err != nil {
		if os.IsNotExist(err) {
			if err = os.MkdirAll(*mountPoint, 0755); err != nil {
				logger.Fatal(err, ", try sudo.")
			}
		}
	}

	// 定义文件系统
	settings.GPFSFilesystem = filepath.Base(*mountPoint)

	minioClient, err := minio.New(settings.MinioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(settings.MinioAccessKey, settings.MinioSecret, ""),
		Region: settings.MinioRegion,
		Secure: false,
	})
	if err != nil {
		logger.Fatalf("Failed to create Minio client: %v", err)
	}
	c, err := fuse.Mount(
		*mountPoint,
		fuse.FSName("gpfs-fuse"),
		fuse.Subtype("customFS"),
		fuse.AllowOther(),
	)
	if err != nil {
		logger.Fatal(err)
	}
	defer func() {
		_ = c.Close()
	}()
	customFS := &services.CustomFS{
		ExternalStorage: *externalStorage,
		MinioClient:     minioClient,
	}
	err = fs.Serve(c, customFS)
	if err != nil {
		logger.Fatal(err)
	}
}
