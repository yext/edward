package cmd

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Display service status",
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.WithStack(edwardClient.Status(args))
	},
}

func init() {
	RootCmd.AddCommand(statusCmd)
}
