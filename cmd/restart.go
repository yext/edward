package cmd

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// restartCmd represents the restart command
var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Rebuild and relaunch a service or a group",
	RunE: func(cmd *cobra.Command, args []string) error {
		err := edwardClient.Restart(
			args,
			*restartFlags.force,
			*restartFlags.skipBuild,
			*restartFlags.noWatch,
			*restartFlags.exclude,
		)
		if err != nil {
			return errors.WithStack(err)
		}
		if *restartFlags.tail {
			return errors.WithStack(edwardClient.Log(args, getSignalChannel()))
		}
		return nil
	},
}

var restartFlags struct {
	skipBuild *bool
	noWatch   *bool
	tail      *bool
	exclude   *[]string
	force     *bool
}

func init() {
	RootCmd.AddCommand(restartCmd)

	restartFlags.skipBuild = restartCmd.Flags().BoolP("skip-build", "s", false, "Skip the build phase")
	restartFlags.noWatch = restartCmd.Flags().Bool("no-watch", false, "Disable autorestart")
	restartFlags.force = restartCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")
	restartFlags.tail = restartCmd.Flags().BoolP("tail", "t", false, "After starting, tail logs for services.")
	restartFlags.exclude = restartCmd.Flags().StringArrayP("exclude", "e", nil, "Exclude `SERVICE` from this operation")
}
