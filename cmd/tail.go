package cmd

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// tailCmd represents the tail command
var tailCmd = &cobra.Command{
	Use:     "tail",
	Short:   "Tail the log for a service",
	Aliases: []string{"log"},
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.WithStack(edwardClient.Log(args, getSignalChannel()))
	},
}

func init() {
	RootCmd.AddCommand(tailCmd)
}

func getSignalChannel() <-chan struct{} {
	sigs := make(chan os.Signal, 1)
	done := make(chan struct{})
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		_ = <-sigs
		close(done)
	}()
	return done
}
