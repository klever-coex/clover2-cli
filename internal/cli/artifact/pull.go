package artifact

import (
	"path/filepath"

	"github.com/minio/minio-go/v7"
	"github.com/spf13/cobra"

	core "github.com/klever-coex/clover2-cli/internal/artifact"
	"github.com/klever-coex/clover2-cli/internal/cliapp"
)

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

			urlEndpoint, urlBucket, key, _ := core.ParseObjectURL(args[0])
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
