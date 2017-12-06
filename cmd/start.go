package cmd

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Build and launch a service or a group",
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.WithStack(
			edwardClient.Start(args, *startFlags.skipBuild, *startFlags.tail, *startFlags.noWatch, *startFlags.exclude),
		)
	},
}

var startFlags struct {
	skipBuild *bool
	noWatch   *bool
	tail      *bool
	exclude   *[]string
}

func init() {
	RootCmd.AddCommand(startCmd)

	startFlags.skipBuild = startCmd.Flags().BoolP("skip-build", "s", false, "Skip the build phase")
	startFlags.noWatch = startCmd.Flags().Bool("no-watch", false, "Disable autorestart")
	startFlags.tail = startCmd.Flags().BoolP("tail", "t", false, "After starting, tail logs for services.")
	startFlags.exclude = startCmd.Flags().StringArrayP("exclude", "e", nil, "Exclude `SERVICE` from this operation")
}
