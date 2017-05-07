package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
	"github.com/yext/edward/common"
	"github.com/yext/edward/config"
	"github.com/yext/edward/generators"
	"github.com/yext/edward/home"
	"github.com/yext/edward/output"
	"github.com/yext/edward/runner"
	"github.com/yext/edward/services"
	"github.com/yext/edward/tracker"
	"github.com/yext/edward/updates"
	"github.com/yext/edward/worker"
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
			Destination: &(flags.config),
		},
	}
	app.Commands = []cli.Command{
		runner.Command,
		{
			Name:   "list",
			Usage:  "List available services",
			Action: list,
		},
		{
			Name:   "generate",
			Usage:  "Generate Edward config for a source tree",
			Action: generate,
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:        "no_prompt, n",
					Usage:       "Skip confirmation prompts",
					Destination: &(flags.noPrompt),
				},
			},
		},
		{
			Name:         "status",
			Usage:        "Display service status",
			Action:       status,
			BashComplete: autocompleteServicesAndGroups,
		},
		{
			Name:         "start",
			Usage:        "Build and launch a service",
			Action:       start,
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
			Name:         "stop",
			Usage:        "Stop a service",
			Action:       stop,
			BashComplete: autocompleteServicesAndGroups,
			Flags: []cli.Flag{
				excludeFlag,
			},
		},
		{
			Name:         "restart",
			Usage:        "Rebuild and relaunch a service",
			Action:       restart,
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
			Name:         "log",
			Aliases:      []string{"tail"},
			Usage:        "Tail the log for a service",
			Action:       doLog,
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

	if len(flags.config) > 0 {
		if absfp, err := filepath.Abs(flags.config); err == nil {
			return absfp
		}
		// TODO: Handle the error from filepath.Abs more effectively
	}

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

func list(c *cli.Context) error {

	groupNames := config.GetAllGroupNames()
	serviceNames := config.GetAllServiceNames()

	sort.Strings(groupNames)
	sort.Strings(serviceNames)

	println("Services and groups")
	println("Groups:")
	for _, name := range groupNames {
		println("\t", name)
	}
	println("Services:")
	for _, name := range serviceNames {
		println("\t", name)
	}

	return nil
}

func generate(c *cli.Context) error {
	var cfg config.Config
	configPath := getConfigPath()
	if configPath == "" {
		wd, err := os.Getwd()
		if err == nil {
			configPath = filepath.Join(wd, "edward.json")
		}
	}

	if _, err := os.Stat(configPath); err == nil {
		r, err := os.Open(configPath)
		if err != nil {
			return errors.WithStack(err)
		}
		cfg, err = config.LoadConfigWithPath(r, configPath, common.EdwardVersion, logger)
		if err != nil {
			return errors.WithMessage(err, configPath)
		}
	} else {
		cfg = config.EmptyConfig(filepath.Dir(configPath), logger)
	}

	wd, err := os.Getwd()
	if err != nil {
		return errors.WithStack(err)
	}

	generators := &generators.GeneratorCollection{
		Generators: []generators.Generator{
			&generators.EdwardGenerator{},
			&generators.DockerGenerator{},
			&generators.GoGenerator{},
			&generators.IcbmGenerator{},
		},
		Path:    wd,
		Targets: c.Args(),
	}
	err = generators.Generate()
	if err != nil {
		return errors.WithStack(err)
	}
	foundServices := generators.Services()
	foundGroups := generators.Groups()
	foundImports := generators.Imports()

	// Prompt user to confirm the list of services that will be generated
	if !flags.noPrompt {
		fmt.Println("The following will be generated:")
		if len(foundServices) > 0 {
			fmt.Println("Services:")
		}
		for _, service := range foundServices {
			fmt.Println("\t", service.Name)
		}
		if len(foundGroups) > 0 {
			fmt.Println("Groups:")
		}
		for _, group := range foundGroups {
			fmt.Println("\t", group.Name)
		}
		if len(foundImports) > 0 {
			fmt.Println("Imports:")
		}
		for _, i := range foundImports {
			fmt.Println("\t", i)
		}

		if !askForConfirmation("Do you wish to continue?") {
			return nil
		}
	}

	foundServices, err = cfg.NormalizeServicePaths(wd, foundServices)
	if err != nil {
		return errors.WithStack(err)
	}
	err = cfg.AppendServices(foundServices)
	if err != nil {
		return errors.WithStack(err)
	}
	err = cfg.AppendGroups(foundGroups)
	if err != nil {
		return errors.WithStack(err)
	}
	cfg.Imports = append(cfg.Imports, foundImports...)

	f, err := os.Create(configPath)
	if err != nil {
		return errors.WithStack(err)
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	err = cfg.Save(w)
	if err != nil {
		return errors.WithStack(err)
	}
	err = w.Flush()
	if err != nil {
		return errors.WithStack(err)
	}

	fmt.Println("Wrote to:", configPath)

	return nil
}

func askForConfirmation(question string) bool {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("%s [y/n]? ", question)

		response, err := reader.ReadString('\n')
		if err != nil {
			return false
		}

		response = strings.ToLower(strings.TrimSpace(response))

		if response == "y" || response == "yes" {
			return true
		} else if response == "n" || response == "no" {
			return false
		}
	}
}

func status(c *cli.Context) error {

	var sgs []services.ServiceOrGroup
	var err error
	if len(c.Args()) == 0 {
		for _, service := range config.GetAllServicesSorted() {
			var s []services.ServiceStatus
			s, err = service.Status()
			if err != nil {
				return errors.WithStack(err)
			}
			for _, status := range s {
				if status.Status != services.StatusStopped {
					sgs = append(sgs, service)
				}
			}
		}
		if len(sgs) == 0 {
			fmt.Println("No services are running")
			return nil
		}
	} else {

		sgs, err = config.GetServicesOrGroups(c.Args())
		if err != nil {
			return errors.WithStack(err)
		}
	}

	if len(sgs) == 0 {
		fmt.Println("No services found")
		return nil
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{
		"Name",
		"Status",
		"PID",
		"Ports",
		"Stdout",
		"Stderr",
		"Start Time",
	})
	table.SetAlignment(tablewriter.ALIGN_LEFT)

	for _, s := range sgs {
		statuses, err := s.Status()
		if err != nil {
			return errors.WithStack(err)
		}
		for _, status := range statuses {
			table.Append([]string{
				status.Service.Name,
				status.Status,
				strconv.Itoa(status.Pid),
				strings.Join(status.Ports, ", "),
				strconv.Itoa(status.StdoutCount) + " lines",
				strconv.Itoa(status.StderrCount) + " lines",
				status.StartTime.Format("2006-01-02 15:04:05"),
			})
		}
	}
	table.Render()
	return nil
}

func start(c *cli.Context) error {
	if len(c.Args()) == 0 {
		return errors.New("At least one service or group must be specified")
	}

	if flags.watch {
		color.Set(color.FgYellow)
		println("The watch flag has been deprecated.\nWatches are now always enabled and run with services in the background.")
		color.Unset()
	}

	sgs, err := config.GetServicesOrGroups(c.Args())
	if err != nil {
		return errors.WithStack(err)
	}
	err = sudoIfNeeded(sgs)
	if err != nil {
		return errors.WithStack(err)
	}

	err = startAndTrack(c, sgs)
	if err != nil {
		return errors.WithStack(err)
	}
	if flags.tail {
		return errors.WithStack(tailFromFlag(c))
	}

	return nil
}

func startAndTrack(c *cli.Context, sgs []services.ServiceOrGroup) error {
	cfg := getOperationConfig()
	err := output.FollowTask(func(t tracker.Task) error {
		p := worker.NewPool(1)
		p.Start()
		defer func() {
			p.Stop()
			_ = <-p.Complete()
		}()
		var err error
		for _, s := range sgs {
			if flags.skipBuild {
				err = s.Launch(cfg, t, p)
			} else {
				err = s.Start(cfg, t, p)
			}
			if err != nil {
				return errors.New("Error launching " + s.GetName() + ": " + err.Error())
			}
		}
		return nil
	})
	return errors.WithStack(err)
}

func stop(c *cli.Context) error {
	var sgs []services.ServiceOrGroup
	var err error
	if len(c.Args()) == 0 {
		allSrv := config.GetAllServicesSorted()
		for _, service := range allSrv {
			var s []services.ServiceStatus
			s, err = service.Status()
			if err != nil {
				return errors.WithStack(err)
			}
			for _, status := range s {
				if status.Status != services.StatusStopped {
					sgs = append(sgs, service)
				}
			}
		}
	} else {
		sgs, err = config.GetServicesOrGroups(c.Args())
		if err != nil {
			return errors.WithStack(err)
		}
	}
	err = sudoIfNeeded(sgs)
	if err != nil {
		return errors.WithStack(err)
	}

	cfg := getOperationConfig()
	err = output.FollowTask(func(t tracker.Task) error {
		p := worker.NewPool(3)
		p.Start()
		defer func() {
			p.Stop()
			_ = <-p.Complete()
		}()
		for _, s := range sgs {
			_ = s.Stop(cfg, t, p)
		}
		return nil
	})

	return errors.WithStack(err)
}

func restart(c *cli.Context) error {
	if len(c.Args()) == 0 {
		restartAll()
	} else {
		err := restartOneOrMoreServices(c.Args())
		if err != nil {
			return errors.WithStack(err)
		}
	}

	if flags.tail {
		return errors.WithStack(tailFromFlag(c))
	}
	return nil
}

func restartAll() error {
	var as []*services.ServiceConfig
	for _, service := range config.GetServiceMap() {
		s, err := service.Status()
		if err != nil {
			return errors.WithStack(err)
		}
		for _, status := range s {
			if status.Status != services.StatusStopped {
				as = append(as, service)
			}
		}
	}

	sort.Sort(serviceConfigByPID(as))
	var serviceNames []string
	for _, service := range as {
		serviceNames = append(serviceNames, service.Name)
	}

	return errors.WithStack(restartOneOrMoreServices(serviceNames))
}

func restartOneOrMoreServices(serviceNames []string) error {
	sgs, err := config.GetServicesOrGroups(serviceNames)
	if err != nil {
		return errors.WithStack(err)
	}
	err = sudoIfNeeded(sgs)
	if err != nil {
		return errors.WithStack(err)
	}

	cfg := getOperationConfig()
	err = output.FollowTask(func(t tracker.Task) error {
		launchPool := worker.NewPool(1)
		launchPool.Start()
		defer func() {
			launchPool.Stop()
			_ = <-launchPool.Complete()
		}()
		for _, s := range sgs {
			stopPool := worker.NewPool(3)
			stopPool.Start()
			err = s.Stop(cfg, t, stopPool)
			if err != nil {
				return errors.WithStack(err)
			}
			stopPool.Stop()
			_ = <-stopPool.Complete()

			if flags.skipBuild {
				err = s.Launch(cfg, t, launchPool)
			} else {
				err = s.Start(cfg, t, launchPool)
			}
			if err != nil {
				return errors.WithStack(err)
			}
		}
		return nil
	})
	return errors.WithStack(err)
}

var flags = struct {
	config    string
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
	}
}

type serviceConfigByPID []*services.ServiceConfig

func (s serviceConfigByPID) Len() int {
	return len(s)
}
func (s serviceConfigByPID) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s serviceConfigByPID) Less(i, j int) bool {
	cmd1, _ := s[i].GetCommand()
	cmd2, _ := s[j].GetCommand()
	return cmd1.Pid < cmd2.Pid
}
