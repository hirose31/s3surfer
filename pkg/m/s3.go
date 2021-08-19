package m

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	s3manager "github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// S3Model ...
type S3Model struct {
	bucket           string
	availableBuckets []string
	prefix           string
	client           *s3.Client
	downloader       *s3manager.Downloader
	cache            map[string]*ObjectCache
}

type ObjectCache struct {
	prefixes []string
	keys     []string
}

func NewS3Model() *S3Model {
	s3m := S3Model{}

	// client
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatal(err)
	}

	s3m.client = s3.NewFromConfig(cfg)
	s3m.downloader = s3manager.NewDownloader(s3m.client)

	// avaiable buckets
	output, err := s3m.client.ListBuckets(context.TODO(), &s3.ListBucketsInput{})
	if err != nil {
		log.Fatal(err)
	}

	for _, bucket := range output.Buckets {
		s3m.availableBuckets = append(s3m.availableBuckets, aws.ToString(bucket.Name))
	}

	if len(s3m.AvailableBuckets()) == 0 {
		log.Fatal("no available S3 buckets")
	}

	// cache
	s3m.cache = map[string]*ObjectCache{}

	return &s3m
}

func (s3m S3Model) Bucket() string {
	return s3m.bucket
}

func (s3m *S3Model) SetBucket(bucket string) error {
	if s3m.bucket != "" {
		return fmt.Errorf("bucket is already set: %s", s3m.bucket)
	}

	found := false
	for _, ab := range s3m.availableBuckets {
		if ab == bucket {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("not found in available buckets: %s", bucket)

	}

	s3m.bucket = bucket
	return nil
}

func (s3m S3Model) AvailableBuckets() []string {
	return s3m.availableBuckets
}

func (s3m S3Model) Prefix() string {
	return s3m.prefix
}

func (s3m *S3Model) setPrefix(prefix string) error {
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		return fmt.Errorf("prefix must be end with slash: %s", prefix)
	}

	s3m.prefix = prefix
	return nil
}

func (s3m *S3Model) MoveUp() error {
	return s3m.setPrefix(upperPrefix((s3m.prefix)))
}

func (s3m *S3Model) MoveDown(prefix string) error {
	return s3m.setPrefix(s3m.prefix + prefix)
}

func (s3m S3Model) List() (
	prefixes []string,
	keys []string,
	err error,
) {
	if s3m.bucket == "" {
		return nil, nil, fmt.Errorf("bucket not set")
	}

	if cache, ok := s3m.cache[s3m.prefix]; ok {
		return cache.prefixes, cache.keys, nil
	}

	input := &s3.ListObjectsV2Input{
		Bucket:    aws.String(s3m.bucket),
		Delimiter: aws.String("/"),
		Prefix:    aws.String(s3m.prefix),
	}

	paginator := s3.NewListObjectsV2Paginator(s3m.client, input)
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(context.TODO())
		if err != nil {
			panic(err)
		}

		for _, prefix := range output.CommonPrefixes {
			prefixes = append(prefixes, lastPartPrefix(aws.ToString(prefix.Prefix)))
		}
		for _, object := range output.Contents {
			keys = append(keys, lastPartPrefix(aws.ToString(object.Key)))
		}
	}

	s3m.cache[s3m.prefix] = &ObjectCache{
		prefixes: prefixes,
		keys:     keys,
	}

	return prefixes, keys, nil
}

func (s3m S3Model) ListObjects(key string) []s3types.Object {
	var objects []s3types.Object

	targetPrefix := s3m.Prefix()
	if key != "" {
		targetPrefix += key
	}

	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(s3m.Bucket()),
		Prefix: aws.String(targetPrefix),
	}

	paginator := s3.NewListObjectsV2Paginator(s3m.client, input)
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(context.TODO())
		if err != nil {
			panic(err)
		}

		for _, object := range output.Contents {
			objects = append(objects, object)
		}
	}

	return objects
}

func (s3m S3Model) Download(object s3types.Object) (n int64, err error) {
	filePath := aws.ToString(object.Key)

	if err = os.MkdirAll(filepath.Dir(filePath), 0700); err != nil {
		return 0, err
	}

	_, err = os.Stat(filePath)
	if err == nil {
		return 0, fmt.Errorf("exists")
	}

	fp, err := os.Create(filePath)
	if err != nil {
		return 0, err
	}
	defer fp.Close()

	return s3m.downloader.Download(
		context.TODO(),
		fp,
		&s3.GetObjectInput{
			Bucket: aws.String(s3m.Bucket()),
			Key:    object.Key,
		},
	)
}

func upperPrefix(prefix string) string {
	if prefix == "" {
		return ""
	}

	prefixNoslash := prefix[:len(prefix)-1]
	i := strings.LastIndex(prefixNoslash, "/")

	if i == -1 {
		// "foo/" => ""
		return ""
	}

	// "foo/bar/baz/" => "foo/bar/"
	return prefixNoslash[:i+1]
}

func lastPartPrefix(prefix string) string {
	if prefix == "" {
		return ""
	}

	prefixNoslash := prefix[:len(prefix)-1]
	i := strings.LastIndex(prefixNoslash, "/")

	if i == -1 {
		// "foo/" => "foo/"
		return prefix
	}

	// "foo/bar/baz/" => "baz/"
	return prefix[i+1:]
}
