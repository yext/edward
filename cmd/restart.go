package cmd

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// restartCmd represents the restart command
var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Rebuild and relaunch a service",
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.WithStack(
			edwardClient.Restart(args, *skipBuild, *tail, *noWatch, *exclude),
		)
	},
}

func init() {
	RootCmd.AddCommand(restartCmd)

	skipBuild = restartCmd.Flags().BoolP("skip-build", "s", false, "Skip the build phase")
	noWatch = restartCmd.Flags().Bool("no-watch", false, "Disable autorestart")
	tail = restartCmd.Flags().BoolP("tail", "t", false, "After starting, tail logs for services.")
	exclude = restartCmd.Flags().StringArrayP("exclude", "e", nil, "Exclude `SERVICE` from this operation")
	timeout = restartCmd.Flags().Int("timeout", 30, "The amount of time in seconds that Edward will wait for a service to launch before timing out. Defaults to 30s")
}
