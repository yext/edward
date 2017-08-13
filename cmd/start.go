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
			edwardClient.Start(args, *skipBuild, *tail, *noWatch, *exclude),
		)
	},
}

var skipBuild *bool
var noWatch *bool
var tail *bool
var exclude *[]string
var timeout *int

func init() {
	RootCmd.AddCommand(startCmd)

	skipBuild = startCmd.Flags().BoolP("skip-build", "s", false, "Skip the build phase")
	noWatch = startCmd.Flags().Bool("no-watch", false, "Disable autorestart")
	tail = startCmd.Flags().BoolP("tail", "t", false, "After starting, tail logs for services.")
	exclude = startCmd.Flags().StringArrayP("exclude", "e", nil, "Exclude `SERVICE` from this operation")
	timeout = startCmd.Flags().Int("timeout", 30, "The amount of time in seconds that Edward will wait for a service to launch before timing out. Defaults to 30s")
}
