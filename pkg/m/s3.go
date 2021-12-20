package m

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	s3manager "github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// S3Bucket ...
type S3Bucket struct {
	Name   string
	Region string
}

type optsFunc = func(*config.LoadOptions) error

// S3Model ...
type S3Model struct {
	bucket           string
	pathStyle        bool
	availableBuckets []S3Bucket
	prefix           string
	client           *s3.Client
	downloader       *s3manager.Downloader
	cache            map[string]*ObjectCache
	endpointURL      string
}

// ObjectCache ...
type ObjectCache struct {
	prefixes []string
	keys     []string
}

// NewS3Model ...
func NewS3Model(endpointURL string, region string, pathStyle bool) *S3Model {
	s3m := S3Model{}

	// client
	if region == "" {
		region = "us-east-1"
		if strings.HasPrefix(os.Getenv("LANG"), "ja") {
			region = "ap-northeast-1"
		}
	}
	opts := []optsFunc{
		config.WithRegion(region),
	}

	if endpointURL != "" {
		endpoint := aws.EndpointResolverFunc(func(service, r string) (aws.Endpoint, error) {
			return aws.Endpoint{
				URL:               endpointURL,
				SigningRegion:     r,
				HostnameImmutable: pathStyle,
			}, nil
		})
		s3m.endpointURL = endpointURL
		opts = append(opts, config.WithEndpointResolver(endpoint))
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(), opts...)
	if err != nil {
		panic(err)
	}

	s3m.client = s3.NewFromConfig(cfg)

	// avaiable buckets
	output, err := s3m.client.ListBuckets(context.TODO(), &s3.ListBucketsInput{})
	if err != nil {
		panic(err)
	}

	for _, bucket := range output.Buckets {
		bl, err := s3m.client.GetBucketLocation(
			context.TODO(),
			&s3.GetBucketLocationInput{
				Bucket: bucket.Name,
			},
		)
		if err != nil {
			panic(err)
		}

		// NormalizeBucketLocation in aws-sd-go v1
		// Replaces empty string with "us-east-1", and "EU" with "eu-west-1".
		//
		// See http://docs.aws.amazon.com/AmazonS3/latest/API/RESTBucketGETlocation.html
		// for more information on the values that can be returned.
		region := string(bl.LocationConstraint)
		switch region {
		case "":
			region = "us-east-1"
		case "EU":
			region = "eu-west-1"
		}
		s3m.availableBuckets = append(s3m.availableBuckets,
			S3Bucket{
				Name:   aws.ToString(bucket.Name),
				Region: region,
			},
		)
	}

	if len(s3m.AvailableBuckets()) == 0 {
		panic("no available S3 buckets")
	}

	// cache
	s3m.cache = map[string]*ObjectCache{}

	s3m.pathStyle = pathStyle

	return &s3m
}

// Bucket ...
func (s3m S3Model) Bucket() string {
	return s3m.bucket
}

// SetBucket ...
func (s3m *S3Model) SetBucket(bucket string) error {
	if s3m.bucket != "" {
		return fmt.Errorf("bucket is already set: %s", s3m.bucket)
	}

	for _, ab := range s3m.AvailableBuckets() {
		if ab.Name != bucket {
			continue
		}

		// found
		s3m.bucket = bucket

		opts := []optsFunc{
			config.WithRegion(ab.Region),
		}
		if s3m.endpointURL != "" {
			endpoint := aws.EndpointResolverFunc(func(service, r string) (aws.Endpoint, error) {
				return aws.Endpoint{
					URL:               s3m.endpointURL,
					SigningRegion:     r,
					HostnameImmutable: s3m.pathStyle,
				}, nil
			})
			opts = append(opts, config.WithEndpointResolver(endpoint))
		}

		// re-create client with region
		cfg, err := config.LoadDefaultConfig(context.TODO(), opts...)
		if err != nil {
			panic(err)
		}

		s3m.client = s3.NewFromConfig(cfg)
		s3m.downloader = s3manager.NewDownloader(s3m.client)

		return nil
	}

	return fmt.Errorf("not found in available buckets: %s", bucket)
}

// AvailableBuckets ...
func (s3m S3Model) AvailableBuckets() []S3Bucket {
	return s3m.availableBuckets
}

// Prefix ...
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

// MoveUp ...
func (s3m *S3Model) MoveUp() error {
	return s3m.setPrefix(upperPrefix((s3m.prefix)))
}

// MoveDown ...
func (s3m *S3Model) MoveDown(prefix string) error {
	return s3m.setPrefix(s3m.prefix + prefix)
}

// List ...
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

// ListObjects ...
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

		objects = append(objects, output.Contents...)
	}

	return objects
}

// Download ...
func (s3m S3Model) Download(object s3types.Object, destPath string) (n int64, err error) {
	if err = os.MkdirAll(filepath.Dir(destPath), 0700); err != nil {
		return 0, err
	}

	_, err = os.Stat(destPath)
	if err == nil {
		return 0, fmt.Errorf("exists")
	}

	fp, err := os.Create(destPath)
	if err != nil {
		return 0, err
	}
	/* #nosec G307 */
	defer func() {
		if err := fp.Close(); err != nil {
			panic(err)
		}
	}()

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
