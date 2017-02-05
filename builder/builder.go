package builder

import (
	"os"
	"os/exec"

	"github.com/pkg/errors"
	"github.com/urfave/cli"
	"github.com/yext/edward/config"
	"github.com/yext/edward/runner"
)

var Command = cli.Command{
	Name:   "build",
	Hidden: true,
	Action: build,
}

func build(c *cli.Context) error {
	if len(c.Args()) == 0 {
		return errors.New("service name is required")
	}

	service, ok := config.GetServiceMap()[c.Args()[0]]
	if !ok {
		return errors.New("service not found")
	}
	if service.Commands.Build == "" {
		return errors.New("no build command configured")
	}

	command, cmdArgs, err := runner.ParseCommand(service.Commands.Build)
	if err != nil {
		return errors.WithStack(err)
	}
	cmd := exec.Command(command, cmdArgs...)
	if service.Path != nil {
		cmd.Dir = *service.Path
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
