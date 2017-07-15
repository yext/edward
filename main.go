package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
	"github.com/yext/edward/common"
	"github.com/yext/edward/config"
	"github.com/yext/edward/edward"
	"github.com/yext/edward/home"
	"github.com/yext/edward/runner"
	"github.com/yext/edward/services"
	"github.com/yext/edward/updates"
)

var logger *log.Logger

func main() {

	if err := home.EdwardConfig.Initialize(); err != nil {
		fmt.Printf("%+v", err)
	}

	logger = log.New(&lumberjack.Logger{
		Filename:   filepath.Join(home.EdwardConfig.EdwardLogDir, "edward.log"),
		MaxSize:    50, // megabytes
		MaxBackups: 30,
		MaxAge:     1, //days
	}, "", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)

	edwardClient := edward.NewClient()
	// Set the default config path
	edwardClient.Config = getConfigPath()
	// Set service checks to restart the client on sudo as needed
	edwardClient.ServiceChecks = func(sgs []services.ServiceOrGroup) error {
		return errors.WithStack(sudoIfNeeded(sgs))
	}
	edwardClient.Logger = logger

	var checkUpdateChan chan interface{}

	app := cli.NewApp()
	app.Name = "Edward"
	app.Usage = "Manage local microservices"
	app.Version = common.EdwardVersion
	app.EnableBashCompletion = true
	app.Before = func(c *cli.Context) error {
		command := c.Args().First()

		if command != "generate" {
			err := config.LoadSharedConfig(getConfigPath(), common.EdwardVersion, logger)
			if err != nil {
				return errors.WithStack(err)
			}
			err = os.Chdir(config.GetBasePath())
			if err != nil {
				return errors.WithStack(err)
			}
		} else {
			config.InitEmptyConfig()
		}

		if command != "stop" {
			// Check for legacy pidfiles and error out if any are found
			for _, service := range config.GetServiceMap() {
				if _, err := os.Stat(service.GetPidPathLegacy()); !os.IsNotExist(err) {
					return errors.New("one or more services were started with an older version of Edward. Please run `edward stop` to stop these instances.")
				}
			}
		}

		if command != "run" {
			checkUpdateChan = make(chan interface{})
			go checkUpdateAvailable(checkUpdateChan)
		}

		return nil
	}

	excludeFlag := cli.StringSliceFlag{
		Name:  "exclude, e",
		Usage: "Exclude `SERVICE` from this operation",
		Value: &(flags.exclude),
	}

	// TODO: This should not be global
	timeoutFlag := cli.IntFlag{
		Name:        "timeout",
		Usage:       "The amount of time in seconds that Edward will wait for a service to launch before timing out. Defaults to 30",
		Destination: &services.StartupTimeoutSeconds,
		Value:       30,
	}

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "config, c",
			Usage:       "Use service configuration file at `PATH`",
			Destination: &(edwardClient.Config),
		},
	}
	app.Commands = []cli.Command{
		runner.Command,
		{
			Name:  "list",
			Usage: "List available services",
			Action: func(c *cli.Context) error {
				err := edwardClient.List()
				return errors.WithStack(err)
			},
		},
		{
			Name:  "generate",
			Usage: "Generate Edward config for a source tree",
			Action: func(c *cli.Context) error {
				return errors.WithStack(
					edwardClient.Generate(c.Args(), flags.noPrompt),
				)
			},
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:        "no_prompt, n",
					Usage:       "Skip confirmation prompts",
					Destination: &(flags.noPrompt),
				},
			},
		},
		{
			Name:  "status",
			Usage: "Display service status",
			Action: func(c *cli.Context) error {
				return errors.WithStack(edwardClient.Status(c.Args()))
			},
			BashComplete: autocompleteServicesAndGroups,
		},
		{
			Name:  "start",
			Usage: "Build and launch a service",
			Action: func(c *cli.Context) error {
				if flags.watch {
					color.Set(color.FgYellow)
					println("The watch flag has been deprecated.\nWatches are now always enabled and run with services in the background.")
					color.Unset()
				}
				err := edwardClient.Start(c.Args(), flags.skipBuild, flags.tail, flags.noWatch, flags.exclude)
				return errors.WithStack(err)
			},
			BashComplete: autocompleteServicesAndGroups,
			Flags: []cli.Flag{
				excludeFlag,
				cli.BoolFlag{
					Name:        "skip-build, s",
					Usage:       "Skip the build phase",
					Destination: &(flags.skipBuild),
				},
				cli.BoolFlag{
					Name:        "watch, w",
					Usage:       "Deprecated, watches are now enabled by default",
					Destination: &(flags.watch),
					Hidden:      true,
				},
				cli.BoolFlag{
					Name:        "no-watch",
					Usage:       "Disable autorestart",
					Destination: &(flags.noWatch),
				},
				cli.BoolFlag{
					Name:        "tail, t",
					Usage:       "After starting, tail logs for services.",
					Destination: &(flags.tail),
				},
				timeoutFlag,
			},
		},
		{
			Name:  "stop",
			Usage: "Stop a service",
			Action: func(c *cli.Context) error {
				return errors.WithStack(edwardClient.Stop(c.Args(), flags.exclude))
			},
			BashComplete: autocompleteServicesAndGroups,
			Flags: []cli.Flag{
				excludeFlag,
			},
		},
		{
			Name:  "restart",
			Usage: "Rebuild and relaunch a service",
			Action: func(c *cli.Context) error {
				return errors.WithStack(
					edwardClient.Restart(c.Args(), flags.skipBuild, flags.tail, flags.noWatch, flags.exclude),
				)
			},
			BashComplete: autocompleteServicesAndGroups,
			Flags: []cli.Flag{
				excludeFlag,
				cli.BoolFlag{
					Name:        "skip-build, s",
					Usage:       "Skip the build phase",
					Destination: &(flags.skipBuild),
				},
				cli.BoolFlag{
					Name:        "tail, t",
					Usage:       "After restarting, tail logs for services.",
					Destination: &(flags.tail),
				},
				cli.BoolFlag{
					Name:        "no-watch",
					Usage:       "Disable autorestart",
					Destination: &(flags.noWatch),
				},
				timeoutFlag,
			},
		},
		{
			Name:    "log",
			Aliases: []string{"tail"},
			Usage:   "Tail the log for a service",
			Action: func(c *cli.Context) error {
				return errors.WithStack(edwardClient.Log(c.Args()))
			},
			BashComplete: autocompleteServicesAndGroups,
		},
	}

	logger.Printf("=== %v v%v ===\n", app.Name, app.Version)
	logger.Printf("Args: %v\n", os.Args)
	defer logger.Printf("=== Exiting ===\n")

	err := app.Run(os.Args)
	if err != nil {
		fmt.Printf("%+v\n", err.Error())
		logger.Printf("%+v", err)
	}

	if checkUpdateChan != nil && !didAutoComplete {
		updateAvailable, ok := (<-checkUpdateChan).(bool)
		if ok && updateAvailable {
			latestVersion := (<-checkUpdateChan).(string)
			fmt.Printf("A new version of Edward is available (%v), update with:\n\tgo get -u github.com/yext/edward\n", latestVersion)
		}
	}

}

// hasBashCompletion identifies whether this call uses the bash completion
// flag.
func hasBashCompletion(c *cli.Context) bool {
	for _, arg := range c.Args() {
		if arg == "--generate-bash-completion" {
			return true
		}
	}
	return false
}

func checkUpdateAvailable(checkUpdateChan chan interface{}) {
	defer close(checkUpdateChan)
	updateAvailable, latestVersion, err := updates.UpdateAvailable("github.com/yext/edward", common.EdwardVersion, filepath.Join(home.EdwardConfig.Dir, ".updatecache"), logger)
	if err != nil {
		logger.Println("Error checking for updates:", err)
		return
	}

	checkUpdateChan <- updateAvailable
	if updateAvailable {
		checkUpdateChan <- latestVersion
	}
}

// getConfigPath identifies the location of edward.json, if any exists
func getConfigPath() string {
	var pathOptions []string

	// Config file in Edward Config dir
	pathOptions = append(pathOptions, filepath.Join(home.EdwardConfig.Dir, "edward.json"))

	// Config file in current working directory
	wd, err := os.Getwd()
	if err == nil {
		pathOptions = append(pathOptions, filepath.Join(wd, "edward.json"))
	}
	for path.Dir(wd) != wd {
		wd = path.Dir(wd)
		pathOptions = append(pathOptions, filepath.Join(wd, "edward.json"))
	}

	for _, path := range pathOptions {
		_, err := os.Stat(path)
		if err != nil {
			continue
		}
		absfp, absErr := filepath.Abs(path)
		if absErr != nil {
			fmt.Println("Error getting config file: ", absErr)
			return ""
		}
		return absfp
	}

	return ""
}

func gitRoot() (string, error) {
	output, err := exec.Command("git", "rev-parse", "--show-toplevel").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%v\n%v", string(output), err)
	}
	return strings.TrimSpace(string(output)), nil
}

var flags = struct {
	skipBuild bool
	watch     bool
	noWatch   bool
	noPrompt  bool
	exclude   cli.StringSlice
	tail      bool
}{}

func getOperationConfig() services.OperationConfig {
	return services.OperationConfig{
		Exclusions: []string(flags.exclude),
		NoWatch:    flags.noWatch,
		SkipBuild:  flags.skipBuild,
	}
}
