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
			edwardClient.Generate(
				args,
				*generateFlags.force || *generateFlags.noPrompt,
				*generateFlags.group,
				*generateFlags.targets,
			),
		)
	},
}

var generateFlags struct {
	force    *bool
	noPrompt *bool
	group    *string
	targets  *[]string
}

func init() {
	RootCmd.AddCommand(generateCmd)

	generateFlags.force = generateCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")

	generateFlags.noPrompt = generateCmd.Flags().BoolP("no_prompt", "n", false, "Skip confirmation prompt")
	generateCmd.Flags().MarkDeprecated("no_prompt", "Please use --force/-f instead.")

	generateFlags.group = generateCmd.Flags().StringP("group", "g", "", "Add newly generated services to a new or existing group.")

	generateFlags.targets = generateCmd.Flags().StringArray("target", nil, "Explicitly specify a target for this generation. If no targets are given, all targets will be used.")
}
