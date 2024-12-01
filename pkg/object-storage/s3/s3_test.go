package s3

import (
	"os"
	"testing"
)

func Test_UploadKey(t *testing.T) {
	s3 := NewS3Client(os.Getenv("TEST_BREW_S3_ENDPOINT"), os.Getenv("TEST_BREW_S3_REGION"), os.Getenv("TEST_BREW_S3_BUCKET"), os.Getenv("TEST_BREW_S3_ACCESS_KEY"), os.Getenv("TEST_BREW_S3_SECRET_KEY"))

	resp, err := s3.GenClientUploadKey("test", "aaa.png")
	if err != nil {
		t.Fatal(err)
	}

	t.Log(resp)
}

func Test_GenGetPreSignKey(t *testing.T) {
	s3 := NewS3Client(os.Getenv("TEST_BREW_S3_ENDPOINT"), os.Getenv("TEST_BREW_S3_REGION"), os.Getenv("TEST_BREW_S3_BUCKET"), os.Getenv("TEST_BREW_S3_ACCESS_KEY"), os.Getenv("TEST_BREW_S3_SECRET_KEY"))

	resp, err := s3.GenGetObjectPreSignURL("")
	if err != nil {
		t.Fatal(err)
	}

	t.Log(resp)
}
