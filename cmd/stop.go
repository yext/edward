package cmd

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// stopCmd represents the stop command
var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop a service or a group",
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.WithStack(edwardClient.Stop(args,
			*stopFlags.force,
			*stopFlags.exclude))
	},
}

var stopFlags struct {
	exclude *[]string
	force   *bool
}

func init() {
	RootCmd.AddCommand(stopCmd)

	stopFlags.exclude = stopCmd.Flags().StringArrayP("exclude", "e", nil, "Exclude `SERVICE` from this operation")
	stopFlags.force = stopCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")
}
