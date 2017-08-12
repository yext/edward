package cmd

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// tailCmd represents the tail command
var tailCmd = &cobra.Command{
	Use:     "tail",
	Short:   "Tail the log for a service",
	Aliases: []string{"log"},
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.WithStack(edwardClient.Log(args))
	},
}

func init() {
	RootCmd.AddCommand(tailCmd)
}
