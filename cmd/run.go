package cmd

import (
	"github.com/spf13/cobra"
	"github.com/yext/edward/runner"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:    "run",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		r := &runner.Runner{}
		r.NoWatch = *runFlags.noWatch
		r.WorkingDir = *runFlags.directory
		r.Logger = logger
		r.Run(args)
		return nil
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
}
