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
			edwardClient.Generate(args, *noPrompt, *targets),
		)
	},
}

var noPrompt *bool
var targets *[]string

func init() {
	RootCmd.AddCommand(generateCmd)

	noPrompt = generateCmd.Flags().BoolP("no_prompt", "n", false, "Skip confirmation prompt")
	targets = generateCmd.Flags().StringArray("target", nil, "Explicitly specify a target for this generation. If no targets are given, all targets will be used.")
}
