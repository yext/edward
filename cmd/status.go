package cmd

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Display service status",
	RunE: func(cmd *cobra.Command, args []string) error {
		output, err := edwardClient.Status(args)
		if err == nil {
			fmt.Print(output)
		}
		return errors.WithStack(err)
	},
}

func init() {
	RootCmd.AddCommand(statusCmd)
}
