package cmd

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available services and groups",
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.WithStack(edwardClient.List())
	},
}

func init() {
	RootCmd.AddCommand(listCmd)
}
