package plugins

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/breeew/brew-api/internal/core"
	"github.com/breeew/brew-api/pkg/object-storage/s3"
)

func Setup(install func(p core.Plugins), mode string) {
	p := provider[mode]
	if p == nil {
		panic("Setup mode not found: " + mode)
	}
	install(p())
}

var provider = map[string]core.SetupFunc{
	"selfhost": func() core.Plugins {
		return newSelfHostMode()
	},
	"saas": func() core.Plugins {
		return newSaaSPlugin()
	},
}

type ObjectStorageDriver struct {
	StaticDomain string    `toml:"static_domain"`
	Driver       string    `toml:"driver"` // default: local
	S3           *S3Config `toml:"s3"`
}

type S3Config struct {
	Bucket    string `toml:"bucket"`
	Region    string `toml:"region"`
	Endpoint  string `toml:"endpoint"`
	AccessKey string `toml:"access_key"`
	SecretKey string `toml:"secret_key"`
}

func setupObjectStorage(cfg ObjectStorageDriver) core.FileStorage {
	var s core.FileStorage
	switch strings.ToLower(cfg.Driver) {
	case "s3":
		s3Cfg := cfg.S3
		s = &S3FileStorage{
			StaticDomain: cfg.StaticDomain,
			S3:           s3.NewS3Client(s3Cfg.Endpoint, s3Cfg.Region, s3Cfg.Bucket, s3Cfg.AccessKey, s3Cfg.SecretKey),
		}
	default:
		s = &LocalFileStorage{
			StaticDomain: cfg.StaticDomain,
		}
	}

	return s
}

type LocalFileStorage struct {
	StaticDomain string
}

func (lfs *LocalFileStorage) GetStaticDomain() string {
	return lfs.StaticDomain
}

func (lfs *LocalFileStorage) GenUploadFileMeta(filePath, fileName string) (core.UploadFileMeta, error) {
	return core.UploadFileMeta{
		FullPath: filepath.Join(filePath, fileName),
		Domain:   lfs.StaticDomain,
	}, nil
}

// SaveFile stores a file on the local file system.
func (lfs *LocalFileStorage) SaveFile(filePath, fileName string, content []byte) error {
	// Check if the directory exists
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		// If the directory doesn't exist, create it
		err := os.MkdirAll(filePath, 0755)
		if err != nil {
			return fmt.Errorf("failed to create directory: %v", err)
		}
	} else if err != nil {
		// If there's an error other than "not exist", return it
		return fmt.Errorf("failed to check directory: %v", err)
	}

	// Save the file
	fullPath := filepath.Join(filePath, fileName)
	err = os.WriteFile(fullPath, content, 0644)
	if err != nil {
		return fmt.Errorf("failed to save file: %v", err)
	}

	return nil
}

// DeleteFile deletes a file from the local file system using the full file path.
func (lfs *LocalFileStorage) DeleteFile(fullFilePath string) error {
	err := os.Remove(fullFilePath)
	if err != nil {
		return fmt.Errorf("failed to delete file: %v", err)
	}
	return nil
}

type S3FileStorage struct {
	StaticDomain string
	*s3.S3
}

func (fs *S3FileStorage) GetStaticDomain() string {
	return fs.StaticDomain
}

func (fs *S3FileStorage) GenUploadFileMeta(filePath, fileName string) (core.UploadFileMeta, error) {
	key, err := fs.S3.GenClientUploadKey(filePath, fileName)
	if err != nil {
		return core.UploadFileMeta{}, err
	}
	return core.UploadFileMeta{
		FullPath:       filepath.Join(filePath, fileName),
		UploadEndpoint: key,
	}, nil
}

// SaveFile stores a file
func (fs *S3FileStorage) SaveFile(filePath, fileName string, content []byte) error {
	return fs.Upload(filePath, fileName, bytes.NewReader(content))
}

// DeleteFile deletes a file
func (fs *S3FileStorage) DeleteFile(fullFilePath string) error {
	return fs.Delete(fullFilePath)
}
