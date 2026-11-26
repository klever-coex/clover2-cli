// Package artifact wires the "clover2 artifact" command group to the
// artifact-store core in internal/artifact.
package artifact

import (
	"fmt"
	"os"

	"github.com/minio/minio-go/v7"
	"github.com/spf13/cobra"

	core "github.com/klever-coex/clover2-cli/internal/artifact"
	"github.com/klever-coex/clover2-cli/internal/cliapp"
)

const endpointHint = "[artifact] endpoint in tooling.json"

// Command returns the "artifact" command group.
func Command() *cobra.Command {
	group := &cobra.Command{
		Use:   "artifact",
		Short: "Push/pull artifacts via S3-compatible storage (MinIO)",
	}

	group.AddCommand(pushCmd(), pullCmd())

	return group
}

func addConnFlags(cmd *cobra.Command) {
	cmd.Flags().String("endpoint", "", "MinIO endpoint host[:port] or URL (env: "+core.EndpointEnv+")")
	cmd.Flags().String("bucket", "", "bucket (default: "+core.DefaultBucket+")")
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
		endpoint = os.Getenv(core.EndpointEnv)
	}

	if endpoint == "" {
		endpoint = cfg.Artifact.Endpoint
	}

	if endpoint == "" {
		return nil, "", "", fmt.Errorf("endpoint is not set (--endpoint, %s or %s)", core.EndpointEnv, endpointHint)
	}

	bucket := bucketFlag
	if bucket == "" {
		bucket = cfg.Artifact.Bucket
	}

	if bucket == "" {
		bucket = core.DefaultBucket
	}

	secure := true
	if cfg.Artifact.Secure != nil {
		secure = *cfg.Artifact.Secure
	}

	if insecure {
		secure = false
	}

	host, sec := core.SplitEndpoint(endpoint, secure)
	client, err := core.Client(host, sec)
	if err != nil {
		return nil, "", "", err
	}

	return client, endpoint, bucket, nil
}
