package artifact

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/spf13/cobra"

	"github.com/klever-coex/clover2-cli/internal/cliapp"
)

const (
	DefaultBucket = "clover2"
	EndpointEnv   = "CLOVER2_S3_ENDPOINT"
	AccessKeyEnv  = "CLOVER2_S3_ACCESS_KEY"
	SecretKeyEnv  = "CLOVER2_S3_SECRET_KEY"
	endpointHint  = "[artifact] endpoint in tooling.json"
)

func Command() *cobra.Command {
	group := &cobra.Command{
		Use:   "artifact",
		Short: "Push/pull artifacts via S3-compatible storage (MinIO)",
	}

	group.AddCommand(pushCmd(), pullCmd())

	return group
}

func addConnFlags(cmd *cobra.Command) {
	cmd.Flags().String("endpoint", "", "MinIO endpoint host[:port] or URL (env: "+EndpointEnv+")")
	cmd.Flags().String("bucket", "", "bucket (default: "+DefaultBucket+")")
	cmd.Flags().Bool("insecure", false, "disable TLS (endpoint without a scheme)")
}

// resolveConn merges flag > URL > env > config precedence and builds the client.
func resolveConn(cmd *cobra.Command, cfg cliapp.Config, urlEndpoint string) (*minio.Client, string, string, error) {
	endpointFlag, _ := cmd.Flags().GetString("endpoint")
	bucketFlag, _ := cmd.Flags().GetString("bucket")
	insecure, _ := cmd.Flags().GetBool("insecure")

	endpoint := endpointFlag
	if endpoint == "" {
		endpoint = urlEndpoint
	}

	if endpoint == "" {
		endpoint = os.Getenv(EndpointEnv)
	}

	if endpoint == "" {
		endpoint = cfg.Artifact.Endpoint
	}

	if endpoint == "" {
		return nil, "", "", fmt.Errorf("endpoint is not set (--endpoint, %s or %s)", EndpointEnv, endpointHint)
	}

	bucket := bucketFlag
	if bucket == "" {
		bucket = cfg.Artifact.Bucket
	}

	if bucket == "" {
		bucket = DefaultBucket
	}

	secure := true
	if cfg.Artifact.Secure != nil {
		secure = *cfg.Artifact.Secure
	}

	if insecure {
		secure = false
	}

	host, sec := splitEndpoint(endpoint, secure)
	client, err := minio.New(host, &minio.Options{
		Creds:  credentials.NewStaticV4(os.Getenv(AccessKeyEnv), os.Getenv(SecretKeyEnv), ""),
		Secure: sec,
	})
	if err != nil {
		return nil, "", "", fmt.Errorf("bad endpoint '%s': %w", host, err)
	}

	return client, endpoint, bucket, nil
}

// splitEndpoint separates an optional URL scheme from the host part.
func splitEndpoint(endpoint string, defaultSecure bool) (string, bool) {
	if i := strings.Index(endpoint, "://"); i >= 0 {
		return endpoint[i+3:], endpoint[:i] == "https"
	}

	return endpoint, defaultSecure
}

// parseObjectURL splits "<scheme>://<host>/<bucket>/<key>" as printed by push.
func parseObjectURL(source string) (endpoint, bucket, key string, ok bool) {
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

func pushCmd() *cobra.Command {
	var tags []string

	cmd := &cobra.Command{
		Use:   "push PATH",
		Short: "Upload a file to the artifact store",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, cfg, err := cliapp.Load()
			if err != nil {
				return err
			}

			client, _, bucket, err := resolveConn(cmd, cfg, "")
			if err != nil {
				return err
			}

			key, _ := cmd.Flags().GetString("key")
			if key == "" {
				key = filepath.Base(args[0])
			}

			objectTags, err := parseTags(tags)
			if err != nil {
				return err
			}

			if _, err := client.FPutObject(cmd.Context(), bucket, key, args[0], minio.PutObjectOptions{UserTags: objectTags}); err != nil {
				return cliapp.ExitErrorf(cliapp.ExitNetwork, "upload failed: %v", err)
			}

			return cliapp.Emit(cmd, cliapp.Payload{
				"bucket": bucket,
				"key":    key,
				"path":   bucket + "/" + key,
			})
		},
	}

	cmd.Flags().String("key", "", "object key (default: file name)")
	cmd.Flags().StringSlice("tag", nil, "object tag KEY=VALUE (repeatable)")
	addConnFlags(cmd)
	cliapp.PayloadFlags(cmd)

	return cmd
}

func pullCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pull SOURCE",
		Short: "Download an object (key, or URL as printed by 'artifact pull')",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, cfg, err := cliapp.Load()
			if err != nil {
				return err
			}

			urlEndpoint, urlBucket, key, _ := parseObjectURL(args[0])
			client, _, bucket, err := resolveConn(cmd, cfg, urlEndpoint)
			if err != nil {
				return err
			}

			if bucketFlag, _ := cmd.Flags().GetString("bucket"); bucketFlag == "" && urlBucket != "" {
				bucket = urlBucket
			}

			output, _ := cmd.Flags().GetString("output")
			if output == "" {
				output = filepath.Base(key)
			}

			if err := client.FGetObject(cmd.Context(), bucket, key, output, minio.GetObjectOptions{}); err != nil {
				return cliapp.ExitErrorf(cliapp.ExitNetwork, "download failed: %v", err)
			}

			return cliapp.Emit(cmd, cliapp.Payload{
				"bucket": bucket,
				"key":    key,
				"path":   output,
			})
		},
	}

	cmd.Flags().String("output", "", "output file (default: object file name)")
	addConnFlags(cmd)
	cliapp.PayloadFlags(cmd)

	return cmd
}

func parseTags(raw []string) (map[string]string, error) {
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
