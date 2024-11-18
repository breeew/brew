package plugins

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/breeew/brew-api/internal/core"
	"github.com/breeew/brew-api/pkg/s3"
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

type LocalFileStorage struct{}

func (lfs *LocalFileStorage) GenUploadFileMeta(filePath, fileName string) (core.UploadFileMeta, error) {
	return core.UploadFileMeta{
		FullPath: filepath.Join(filePath, fileName),
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
	*s3.S3
}

func (fs *S3FileStorage) GenUploadFileMeta(filePath, fileName string) (core.UploadFileMeta, error) {
	key, err := fs.S3.GenClientUploadKey(filePath, fileName)
	if err != nil {
		return core.UploadFileMeta{}, err
	}
	return core.UploadFileMeta{
		FullPath: filepath.Join(filePath, fileName),
		Endpoint: key,
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
