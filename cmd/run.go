package cmd

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/yext/edward/common"
	"github.com/yext/edward/config"
	"github.com/yext/edward/runner"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:    "run",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath, err := config.GetConfigPathFromWorkingDirectory()
		if err != nil {
			return errors.WithStack(err)
		}
		cfg, err := config.LoadConfig(configPath, common.EdwardVersion, logger)
		if err != nil {
			return errors.WithMessage(err, configPath)
		}

		service := cfg.ServiceMap[args[0]]
		if service == nil {
			return fmt.Errorf("service not found: %s", args[0])
		}
		r := &runner.Runner{
			Service: service,
		}
		r.NoWatch = *runFlags.noWatch
		r.WorkingDir = *runFlags.directory
		r.Logger = logger
		r.Run(args)
		return nil
	},
}

var runFlags struct {
	noWatch   *bool
	directory *string
}

func init() {
	RootCmd.AddCommand(runCmd)

	runFlags.noWatch = runCmd.Flags().Bool("no-watch", false, "Disable autorestart")
	runFlags.directory = runCmd.Flags().StringP("directory", "d", "", "Working directory")
	_ = runCmd.Flags().StringArrayP("tag", "t", nil, "Tags to distinguish this instance of Edward")
}
