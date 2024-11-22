package s3

import (
	"context"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3 struct {
	Endpoint string
	Region   string
	Bucket   string
	ak       string
	sk       string
}

func NewS3Client(endpoint, region, bucket, ak, sk string) *S3 {
	return &S3{
		Endpoint: endpoint,
		Region:   region,
		Bucket:   bucket,
		ak:       ak,
		sk:       sk,
	}
}

func (s *S3) DefaultConfig(ctx context.Context) (aws.Config, error) {
	return config.LoadDefaultConfig(
		ctx,
		config.WithCredentialsProvider(credentials.StaticCredentialsProvider{
			Value: aws.Credentials{
				AccessKeyID: s.ak, SecretAccessKey: s.sk,
			},
		}),
		config.WithRegion(s.Region),
		config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{
				URL:           s.Endpoint,
				SigningRegion: s.Region,
			}, nil
		})))
}

func (s *S3) GenGetObjectPreSignURL(filePath string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	cfg, err := s.DefaultConfig(ctx)
	if err != nil {
		return "", err
	}
	s3Client := s3.NewFromConfig(cfg)
	s3PresignClient := s3.NewPresignClient(s3Client)
	req, err := s3PresignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String(filePath),
	}, s3.WithPresignExpires(time.Minute))
	if err != nil {
		return "", err
	}

	return req.URL, nil
}

func (s *S3) GenClientUploadKey(filePath, file string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	filePath = strings.TrimPrefix(filePath, "/")

	cfg, err := s.DefaultConfig(ctx)
	if err != nil {
		return "", err
	}
	s3Client := s3.NewFromConfig(cfg)
	s3PresignClient := s3.NewPresignClient(s3Client)
	req, err := s3PresignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String(filepath.Join(filePath, file)),
	}, s3.WithPresignExpires(20*time.Second))
	if err != nil {
		return "", err
	}

	return req.URL, nil
}

func (s *S3) Upload(filePath, file string, body io.Reader) error {
	filePath = strings.TrimPrefix(filePath, "/")
	cfg, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithCredentialsProvider(credentials.StaticCredentialsProvider{
			Value: aws.Credentials{
				AccessKeyID: s.ak, SecretAccessKey: s.sk,
			},
		}),
		config.WithRegion(s.Region),
		config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{
				URL: s.Endpoint,
			}, nil
		})))
	if err != nil {
		return err
	}
	s3Client := s3.NewFromConfig(cfg)
	s3Manager := manager.NewUploader(s3Client)

	_, err = s3Manager.Upload(context.Background(), &s3.PutObjectInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String(filepath.Join(filePath, file)),
		Body:   body,
	})
	if err != nil {
		return err
	}
	return nil
}

func (s *S3) Delete(fullPath string) error {
	cfg, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithCredentialsProvider(credentials.StaticCredentialsProvider{
			Value: aws.Credentials{
				AccessKeyID: s.ak, SecretAccessKey: s.sk,
			},
		}),
		config.WithRegion(s.Region),
		config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{
				URL: s.Endpoint,
			}, nil
		})))
	if err != nil {
		return err
	}
	s3Client := s3.NewFromConfig(cfg)
	_, err = s3Client.DeleteObject(context.Background(), &s3.DeleteObjectInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String(fullPath),
	})
	if err != nil {
		return err
	}
	return nil
}
