package cmd

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/yext/edward/runner"
	"github.com/yext/edward/services"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:    "run",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		service := edwardClient.ServiceMap()[args[0]]
		if service == nil {
			return fmt.Errorf("service not found: %s", args[0])
		}
		c := edwardClient
		r, err := runner.NewRunner(
			services.OperationConfig{
				WorkingDir:       c.WorkingDir,
				EdwardExecutable: c.EdwardExecutable,
				Tags:             c.Tags,
				LogFile:          c.LogFile,
				Backends:         c.Backends,
			},
			service,
			edwardClient.DirConfig,
			*runFlags.noWatch,
			*runFlags.directory,
			edwardClient.Logger,
		)
		if err != nil {
			return errors.WithStack(err)
		}

		return errors.WithStack(
			r.Run(args),
		)
	},
}

var runFlags struct {
	noWatch   *bool
	directory *string
}

func init() {
	RootCmd.AddCommand(runCmd)

	runFlags.noWatch = runCmd.Flags().Bool("no-watch", false, "Disable autorestart")
	runFlags.directory = runCmd.Flags().StringP("directory", "d", "", "Working directory")
	_ = runCmd.Flags().StringArrayP("tag", "t", nil, "Tags to distinguish this instance of Edward")
}
