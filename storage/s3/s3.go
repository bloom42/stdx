package s3

import (
	"context"
	"errors"
	"io"
	"net/http"
	"path/filepath"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type S3Storage struct {
	basePath string
	s3Client *s3.S3
	bucket   string
}

type Config struct {
	AccessKeyID     string
	SecretAccessKey string
	Endpoint        string
	Region          string
	BaseDirectory   string
	Bucket          string
	HttpClient      *http.Client
}

func NewS3Storage(config Config) (*S3Storage, error) {
	var endpointResolver endpoints.Resolver

	if config.Endpoint != "" {
		// see https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/endpoints/
		// https://stackoverflow.com/questions/67575681/is-aws-go-sdk-v2-integrated-with-local-minio-server
		// https://stackoverflow.com/questions/71088064/how-can-i-use-the-aws-sdk-v2-for-go-with-digitalocean-spaces
		endpointResolver = endpoints.ResolverFunc(func(service, region string, optFns ...func(*endpoints.Options)) (endpoints.ResolvedEndpoint, error) {
			return endpoints.ResolvedEndpoint{
				SigningRegion: config.Region,
				URL:           config.Endpoint,
			}, nil
		})
	}

	awsSession, err := session.NewSession(&aws.Config{
		Endpoint:         aws.String(config.Endpoint),
		Region:           aws.String(config.Region),
		Credentials:      credentials.NewStaticCredentials(config.AccessKeyID, config.SecretAccessKey, ""),
		HTTPClient:       config.HttpClient,
		EndpointResolver: endpointResolver,
	})
	if err != nil {
		return nil, err
	}

	// Create S3 service client
	s3Client := s3.New(awsSession)

	return &S3Storage{
		basePath: config.BaseDirectory,
		s3Client: s3Client,
		bucket:   config.Bucket,
	}, nil
}

func (storage *S3Storage) BasePath() string {
	return storage.basePath
}

func (storage *S3Storage) CopyObject(ctx context.Context, from string, to string) error {
	from = filepath.Join(storage.bucket, storage.basePath, from)
	to = filepath.Join(storage.basePath, from)

	_, err := storage.s3Client.CopyObject(&s3.CopyObjectInput{
		Bucket:     aws.String(storage.bucket),
		Key:        aws.String(to),
		CopySource: aws.String(from),
	})
	if err != nil {
		return err
	}

	return nil
}

func (storage *S3Storage) DeleteObject(ctx context.Context, key string) error {
	objectKey := filepath.Join(storage.basePath, key)

	_, err := storage.s3Client.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(storage.bucket),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		return err
	}

	return nil
}

func (storage *S3Storage) GetObject(ctx context.Context, key string) (io.ReadCloser, error) {
	objectKey := filepath.Join(storage.basePath, key)

	result, err := storage.s3Client.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(storage.bucket),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		return nil, err
	}

	return result.Body, nil
}

func (storage *S3Storage) GetObjectSize(ctx context.Context, key string) (int64, error) {
	objectKey := filepath.Join(storage.basePath, key)

	result, err := storage.s3Client.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(storage.bucket),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		return 0, err
	}

	if result.ContentLength == nil {
		return 0, errors.New("s3: object size is null")
	}

	return *result.ContentLength, nil
}

// func (storage *S3Storage) GetPresignedUploadUrl(ctx context.Context, key string, size uint64) (string, error) {
// 	objectKey := filepath.Join(storage.basePath, key)

// 	req, _ := storage.s3Client.PutObjectRequest(&s3.PutObjectInput{
// 		Bucket:        aws.String(storage.bucket),
// 		Key:           aws.String(objectKey),
// 		ContentLength: aws.Int64(int64(size)),
// 	})

// 	url, err := req.Presign(2 * time.Hour)
// 	if err != nil {
// 		return "", err
// 	}

// 	return url, nil
// }

// TODO?: https://docs.aws.amazon.com/AmazonS3/latest/userguide/checking-object-integrity.html
func (storage *S3Storage) PutObject(ctx context.Context, key string, contentType string, size int64, object io.Reader) error {
	objectKey := filepath.Join(storage.basePath, key)

	_, err := storage.s3Client.PutObject(&s3.PutObjectInput{
		Bucket:        aws.String(storage.bucket),
		Key:           aws.String(objectKey),
		Body:          aws.ReadSeekCloser(object),
		ContentType:   aws.String(contentType),
		ContentLength: aws.Int64(int64(size)),
	})
	if err != nil {
		return err
	}

	return nil
}

func (storage *S3Storage) DeleteObjectsWithPrefix(ctx context.Context, prefix string) (err error) {
	s3Prefix := filepath.Join(storage.basePath, prefix)
	var continuationToken *string

	for {
		var res *s3.ListObjectsV2Output

		res, err = storage.s3Client.ListObjectsV2(&s3.ListObjectsV2Input{
			Bucket:            aws.String(storage.bucket),
			Prefix:            aws.String(s3Prefix),
			ContinuationToken: continuationToken,
		})

		for _, object := range res.Contents {
			err = storage.DeleteObject(ctx, *object.Key)
			if err != nil {
				return
			}
		}

		continuationToken = res.ContinuationToken

		if continuationToken == nil {
			break
		}
	}

	return
}
