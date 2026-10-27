package artifact

import (
	"fmt"
	"os"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const (
	DefaultBucket = "clover2"
	EndpointEnv   = "CLOVER2_S3_ENDPOINT"
	AccessKeyEnv  = "CLOVER2_S3_ACCESS_KEY"
	SecretKeyEnv  = "CLOVER2_S3_SECRET_KEY"
)

func Client(host string, secure bool) (*minio.Client, error) {
	client, err := minio.New(host, &minio.Options{
		Creds:  credentials.NewStaticV4(os.Getenv(AccessKeyEnv), os.Getenv(SecretKeyEnv), ""),
		Secure: secure,
	})
	if err != nil {
		return nil, fmt.Errorf("bad endpoint '%s': %w", host, err)
	}

	return client, nil
}

func SplitEndpoint(endpoint string, defaultSecure bool) (string, bool) {
	if i := strings.Index(endpoint, "://"); i >= 0 {
		return endpoint[i+3:], endpoint[:i] == "https"
	}

	return endpoint, defaultSecure
}

func ParseObjectURL(source string) (endpoint, bucket, key string, ok bool) {
	i := strings.Index(source, "://")
	if i < 0 {
		return "", "", source, false
	}

	host, rest, _ := strings.Cut(source[i+3:], "/")
	bucket, key, _ = strings.Cut(rest, "/")
	if host == "" || bucket == "" || key == "" {
		return "", "", "", false
	}

	return source[:i] + "://" + host, bucket, key, true
}

func ParseTags(raw []string) (map[string]string, error) {
	tags := map[string]string{}
	for _, item := range raw {
		name, value, ok := strings.Cut(item, "=")
		if !ok || name == "" {
			return nil, fmt.Errorf("invalid tag '%s'; expected KEY=VALUE", item)
		}

		tags[name] = value
	}

	return tags, nil
}
