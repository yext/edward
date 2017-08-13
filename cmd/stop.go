package cmd

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// stopCmd represents the stop command
var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop a service",
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.WithStack(edwardClient.Stop(args, *exclude))
	},
}

func init() {
	RootCmd.AddCommand(stopCmd)

	exclude = stopCmd.Flags().StringArrayP("exclude", "e", nil, "Exclude `SERVICE` from this operation")
}
