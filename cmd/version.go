package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yext/edward/common"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Displays the currently installed version of Edward",
	// Skip loading config
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Edward version %v\n", common.EdwardVersion)
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}
