package cmd

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// tipLogCmd represents the tiplog command
var tipLogCmd = &cobra.Command{
	Use:   "tiplog",
	Short: "View the tip (last 5 lines) of multiple, or all services",
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.WithStack(edwardClient.TipLog(args, 5))
	},
}

func init() {
	RootCmd.AddCommand(tipLogCmd)
}
