package main

import (
    "context"
    "errors"
    "fmt"
    "io"
    "os"
    "path/filepath"

    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/credentials"
    "github.com/aws/aws-sdk-go-v2/service/s3"
    "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// NewS3Client creates a new S3 client with the given config.
func NewS3Client(cfg S3Config) (*s3.Client, error) {
    customResolver := aws.EndpointResolverWithOptionsFunc(
        func(service, region string, options ...interface{}) (aws.Endpoint, error) {
            return aws.Endpoint{
                URL:           cfg.Endpoint,
                SigningRegion: cfg.Region,
                HostnameImmutable: true,
            }, nil
        },
    )
    awsCfg, err := config.LoadDefaultConfig(context.TODO(),
        config.WithRegion(cfg.Region),
        config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, "")),
        config.WithEndpointResolverWithOptions(customResolver),
    )
    if err != nil {
        return nil, err
    }
    return s3.NewFromConfig(awsCfg, func(o *s3.Options) {
        o.UsePathStyle = true
    }), nil
}

// DownloadFile downloads an object from S3 to a local file.
func DownloadFile(ctx context.Context, client *s3.Client, bucket, key, localPath string) error {
    if bucket == "" || key == "" || localPath == "" {
        return errors.New("invalid arguments: bucket, key, and localPath are required")
    }

    out, err := client.GetObject(ctx, &s3.GetObjectInput{
        Bucket: aws.String(bucket),
        Key:    aws.String(key),
    })
    if err != nil {
        var nsk *types.NoSuchKey
        if errors.As(err, &nsk) {
            return fmt.Errorf("no such key: %s in bucket: %s", key, bucket)
        }
        return fmt.Errorf("failed to get object: %w", err)
    }
    defer out.Body.Close()

    // Ensure the directory exists
    if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
        return err
    }

    f, err := os.Create(localPath)
    if err != nil {
        return err
    }
    defer f.Close()

    _, err = io.Copy(f, out.Body)
    return err
}

// UploadFile uploads a local file to S3.
func UploadFile(ctx context.Context, client *s3.Client, bucket, localPath, key string) error {
    if bucket == "" || localPath == "" || key == "" {
        return errors.New("invalid arguments: bucket, localPath, and key are required")
    }

    f, err := os.Open(localPath)
    if err != nil {
        return fmt.Errorf("file not found: %s", localPath)
    }
    defer f.Close()

    _, err = client.PutObject(ctx, &s3.PutObjectInput{
        Bucket: aws.String(bucket),
        Key:    aws.String(key),
        Body:   f,
    })
    if err != nil {
        return fmt.Errorf("failed to upload file: %s to bucket: %s with key: %s. Error: %w", localPath, bucket, key, err)
    }
    return nil
}