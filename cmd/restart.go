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
		return errors.WithStack(
			edwardClient.Restart(
				args,
				*restartFlags.force,
				*restartFlags.skipBuild,
				*restartFlags.tail,
				*restartFlags.noWatch,
				*restartFlags.exclude,
			),
		)
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
