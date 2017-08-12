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
		r.NoWatch = *noWatch
		r.WorkingDir = *runnerDirectory
		r.Logger = logger
		r.Run(args)
		return nil
	},
}

var runnerDirectory *string

func init() {
	RootCmd.AddCommand(runCmd)

	noWatch = runCmd.Flags().Bool("no-watch", false, "Disable autorestart")
	runnerDirectory = runCmd.Flags().StringP("directory", "d", "", "Working directory")
}
