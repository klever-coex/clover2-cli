package artifact

import (
	"path/filepath"

	"github.com/minio/minio-go/v7"
	"github.com/spf13/cobra"

	core "github.com/klever-coex/clover2-cli/internal/artifact"
	"github.com/klever-coex/clover2-cli/internal/cliapp"
)

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

			objectTags, err := core.ParseTags(tags)
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
