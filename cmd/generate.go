package cmd

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// generateCmd represents the generate command
var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Automatically generate Edward config for a source tree",
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.WithStack(
			edwardClient.Generate(args, *noPrompt),
		)
	},
}

var noPrompt *bool

func init() {
	RootCmd.AddCommand(generateCmd)

	noPrompt = generateCmd.Flags().BoolP("no_prompt", "n", false, "Skip confirmation prompt")
}
