package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/hpcloud/tail"
	"github.com/olekukonko/tablewriter"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
	"github.com/yext/edward/config"
	"github.com/yext/edward/generators"
	"github.com/yext/edward/home"
	"github.com/yext/edward/services"
	"github.com/yext/edward/servicewatch"
	"github.com/yext/edward/updates"
	"github.com/yext/edward/warmup"
)

var logger *log.Logger

const edwardVersion = "1.6.4"

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

	app := cli.NewApp()
	app.Name = "Edward"
	app.Usage = "Manage local microservices"
	app.Version = edwardVersion
	app.EnableBashCompletion = true
	app.Before = func(c *cli.Context) error {
		command := c.Args().First()

		if command != "generate" {
			err := loadConfig()
			if err != nil {
				return errors.WithStack(err)
			}
		} else {
			initEmptyConfig()
		}

		return nil
	}

	excludeFlag := cli.StringSliceFlag{
		Name:  "exclude, e",
		Usage: "Exclude `SERVICE` from this operation",
		Value: &(flags.exclude),
	}

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "config, c",
			Usage:       "Use service configuration file at `PATH`",
			Destination: &(flags.config),
		},
	}
	app.Commands = []cli.Command{
		runnerCommand,
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
			Name:         "messages",
			Usage:        "Show messages from services",
			Action:       messages,
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
					Usage:       "After starting, watch services for changes and hot-reload.",
					Destination: &(flags.watch),
				},
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
			},
		},
		{
			Name:         "log",
			Aliases:      []string{"tail"},
			Usage:        "Tail the log for a service",
			Action:       doLog,
			BashComplete: autocompleteServices,
		},
	}

	logger.Printf("=== %v v%v ===\n", app.Name, app.Version)
	logger.Printf("Args: %v\n", os.Args)
	defer logger.Printf("=== Exiting ===\n")

	checkUpdateChan := make(chan interface{})
	go checkUpdateAvailable(checkUpdateChan)

	err := app.Run(os.Args)
	if err != nil {
		fmt.Printf("%+v", err)
		logger.Printf("%+v", err)
	}

	warmup.Wait()

	updateAvailable, ok := (<-checkUpdateChan).(bool)
	if ok && updateAvailable {
		latestVersion := (<-checkUpdateChan).(string)
		fmt.Printf("A new version of Edward is available (%v), update with:\n\tgo get -u github.com/yext/edward\n", latestVersion)
	}

}

func checkUpdateAvailable(checkUpdateChan chan interface{}) {
	defer close(checkUpdateChan)
	updateAvailable, latestVersion, err := updates.UpdateAvailable("github.com/yext/edward", edwardVersion, filepath.Join(home.EdwardConfig.Dir, ".updatecache"), logger)
	if err != nil {
		logger.Println("Error checking for updates:", err)
		return
	}

	checkUpdateChan <- updateAvailable
	if updateAvailable {
		checkUpdateChan <- latestVersion
	}
}

var groupMap map[string]*services.ServiceGroupConfig
var serviceMap map[string]*services.ServiceConfig

// getConfigPath identifies the location of edward.json, if any exists
func getConfigPath() string {

	// TODO: Handle abs path not working more cleanly

	if len(flags.config) > 0 {
		if absfp, err := filepath.Abs(flags.config); err == nil {
			return absfp
		}
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
		if _, err := os.Stat(path); err == nil {
			if absfp, err := filepath.Abs(path); err == nil {
				return absfp
			} else {
				fmt.Println("Error getting config file: ", err)
			}
		}
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

func initEmptyConfig() {
	groupMap = make(map[string]*services.ServiceGroupConfig)
	serviceMap = make(map[string]*services.ServiceConfig)
}

func loadConfig() error {
	initEmptyConfig()

	configPath := getConfigPath()
	if configPath != "" {
		r, err := os.Open(configPath)
		if err != nil {
			return errors.WithStack(err)
		}
		cfg, err := config.LoadConfigWithDir(r, filepath.Dir(configPath), edwardVersion, logger)
		if err != nil {
			return errors.WithStack(err)
		}

		serviceMap = cfg.ServiceMap
		groupMap = cfg.GroupMap
		return nil
	}

	return errors.New("No config file found")

}

func sudoIfNeeded(sgs []services.ServiceOrGroup) error {
	for _, sg := range sgs {
		if sg.IsSudo(getOperationConfig()) {
			logger.Printf("sudo required for %v\n", sg.GetName())
			return errors.WithStack(prepareForSudo())
		}
	}
	logger.Printf("sudo not required for any services/groups\n")
	return nil
}

func getServicesOrGroups(names []string) ([]services.ServiceOrGroup, error) {
	var outSG []services.ServiceOrGroup
	for _, name := range names {
		sg, err := getServiceOrGroup(name)
		if err != nil {
			return nil, err
		}
		outSG = append(outSG, sg)
	}
	return outSG, nil
}

func getServiceOrGroup(name string) (services.ServiceOrGroup, error) {
	if group, ok := groupMap[name]; ok {
		return group, nil
	}
	if service, ok := serviceMap[name]; ok {
		return service, nil
	}
	return nil, errors.New("Service or group not found")
}

func getAllServices() []string {
	var serviceNames []string
	for name := range serviceMap {
		serviceNames = append(serviceNames, name)
	}
	return serviceNames
}

func getAllGroups() []string {
	var groupNames []string
	for name := range groupMap {
		groupNames = append(groupNames, name)
	}
	return groupNames

}

func autocompleteServices(c *cli.Context) {
	loadConfig()
	names := getAllServices()
	for _, name := range names {
		fmt.Println(name)
	}
}

func autocompleteServicesAndGroups(c *cli.Context) {
	loadConfig()
	names := append(getAllGroups(), getAllServices()...)
	for _, name := range names {
		fmt.Println(name)
	}
}

func list(c *cli.Context) error {

	groupNames := getAllGroups()
	serviceNames := getAllServices()

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
	var err error

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
		cfg, err = config.LoadConfigWithDir(r, filepath.Dir(configPath), edwardVersion, logger)
		if err != nil {
			return errors.WithStack(err)
		}
	} else {
		cfg = config.EmptyConfig(filepath.Dir(configPath), logger)
	}

	wd, err := os.Getwd()
	if err != nil {
		return errors.WithStack(err)
	}

	foundServices, serviceToGenerator, err := generators.GenerateServices(wd, c.Args())
	if err != nil {
		return errors.WithStack(err)
	}

	// Prompt user to confirm the list of services that will be generated
	if !flags.noPrompt {
		fmt.Println("The following list of services will be generated:")
		var goServices []string
		var icbmServices []string
		for _, service := range foundServices {
			switch serviceToGenerator[service.Name] {
			case "go":
				goServices = append(goServices, service.Name)
			case "icbm":
				icbmServices = append(icbmServices, service.Name)
			default:
				fmt.Println("\t", service.Name)
			}
		}

		if len(goServices) > 0 {
			fmt.Println("GO:")
		}
		for _, service := range goServices {
			fmt.Println("\t", service)
		}

		if len(icbmServices) > 0 {
			fmt.Println("ICBM:")
		}
		for _, service := range icbmServices {
			fmt.Println("\t", service)
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
		allSrv := allServices()
		for _, service := range allSrv {
			s, err := service.Status()
			if err != nil {
				return errors.WithStack(err)
			}
			for _, status := range s {
				if status.Status != "STOPPED" {
					sgs = append(sgs, service)
				}
			}
		}
		if len(sgs) == 0 {
			fmt.Println("No services are running")
			return nil
		}
	} else {

		sgs, err = getServicesOrGroups(c.Args())
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
				status.StartTime.String(),
			})
		}
	}
	table.Render()
	return nil
}

func messages(c *cli.Context) error {
	return errors.New("Unimplemented")
}

func start(c *cli.Context) error {
	sgs, err := getServicesOrGroups(c.Args())
	if err != nil {
		return errors.WithStack(err)
	}
	err = sudoIfNeeded(sgs)
	if err != nil {
		return errors.WithStack(err)
	}

	for _, s := range sgs {
		if flags.skipBuild {
			err = s.Launch(getOperationConfig())
		} else {
			err = s.Start(getOperationConfig())
		}
		if err != nil {
			return errors.New("Error launching " + s.GetName() + ": " + err.Error())
		}
	}

	if flags.watch {
		println("==== Watch ====")
		return errors.WithStack(servicewatch.Begin(sgs, getOperationConfig()))
	}

	return nil
}

func allServices() []services.ServiceOrGroup {
	var as []services.ServiceOrGroup
	for _, service := range serviceMap {
		as = append(as, service)
	}
	sort.Sort(serviceOrGroupByName(as))
	return as
}

func stop(c *cli.Context) error {
	var sgs []services.ServiceOrGroup
	var err error
	if len(c.Args()) == 0 {
		allSrv := allServices()
		for _, service := range allSrv {
			s, err := service.Status()
			if err != nil {
				return errors.WithStack(err)
			}
			for _, status := range s {
				if status.Status != "STOPPED" {
					sgs = append(sgs, service)
				}
			}
		}
	} else {
		sgs, err = getServicesOrGroups(c.Args())
		if err != nil {
			return errors.WithStack(err)
		}
	}
	err = sudoIfNeeded(sgs)
	if err != nil {
		return errors.WithStack(err)
	}

	for _, s := range sgs {
		_ = s.Stop(getOperationConfig())
	}
	return nil
}

func restart(c *cli.Context) error {
	if len(c.Args()) == 0 {
		restartAll()
		return nil
	}
	return errors.WithStack(restartOneOrMoreServices(c.Args()))
}

func restartAll() error {

	var as []*services.ServiceConfig
	for _, service := range serviceMap {
		s, err := service.Status()
		if err != nil {
			return errors.WithStack(err)
		}
		for _, status := range s {
			if status.Status != "STOPPED" {
				as = append(as, service)
			}
		}
	}

	sort.Sort(ServiceConfigByPID(as))
	var serviceNames []string
	for _, service := range as {
		serviceNames = append(serviceNames, service.Name)
	}

	return errors.WithStack(restartOneOrMoreServices(serviceNames))
}

func restartOneOrMoreServices(serviceNames []string) error {
	sgs, err := getServicesOrGroups(serviceNames)
	if err != nil {
		return errors.WithStack(err)
	}
	err = sudoIfNeeded(sgs)
	if err != nil {
		return errors.WithStack(err)
	}
	for _, s := range sgs {
		err = s.Stop(getOperationConfig())
		if err != nil {
			return errors.WithStack(err)
		}
		if flags.skipBuild {
			err = s.Launch(getOperationConfig())
		} else {
			err = s.Start(getOperationConfig())
		}
		if err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

func doLog(c *cli.Context) error {
	sgs, err := getServicesOrGroups(c.Args())
	if err != nil {
		return errors.WithStack(err)
	}

	count := services.CountServices(sgs)

	var wait sync.WaitGroup

	for _, sg := range sgs {
		switch v := sg.(type) {
		case *services.ServiceConfig:
			followServiceLog(v, count > 1, &wait)
		case *services.ServiceGroupConfig:
			followGroupLog(v, count > 1, &wait)
		}
	}

	wait.Wait()

	return nil
}

func followGroupLog(group *services.ServiceGroupConfig, multiple bool, wait *sync.WaitGroup) error {
	for _, group := range group.Groups {
		followGroupLog(group, multiple, wait)
	}
	for _, service := range group.Services {
		followServiceLog(service, multiple, wait)
	}
	return nil
}

func followServiceLog(service *services.ServiceConfig, multiple bool, wait *sync.WaitGroup) error {
	wait.Add(1)
	go doFollowServiceLog(service, multiple, wait)
	return nil
}

func doFollowServiceLog(service *services.ServiceConfig, multiple bool, wait *sync.WaitGroup) error {
	command, err := service.GetCommand()
	if err != nil {
		return errors.WithStack(err)
	}
	runLog := command.Scripts.Launch.Log
	t, err := tail.TailFile(runLog, tail.Config{Follow: true})
	if err != nil {
		return errors.WithStack(err)
	}
	for line := range t.Lines {
		var lineData LogLine
		err = json.Unmarshal([]byte(line.Text), &lineData)
		if err != nil {
			return errors.WithStack(err)
		}
		if multiple {
			fmt.Printf("%v: %v\n", service.Name, lineData.Message)
		} else {
			fmt.Printf("%v\n", lineData.Message)
		}
	}
	wait.Done()
	return nil
}

func checkNotSudo() error {
	user, err := user.Current()
	if err != nil {
		return errors.WithStack(err)
	}
	if user.Uid == "0" {
		return errors.New("edward should not be fun with sudo")
	}
	return nil
}

func createScriptFile(suffix string, content string) (*os.File, error) {
	file, err := ioutil.TempFile(os.TempDir(), suffix)
	if err != nil {
		return nil, err
	}
	file.WriteString(content)
	file.Close()

	err = os.Chmod(file.Name(), 0777)
	if err != nil {
		return nil, err
	}

	return file, nil
}

func ensureSudoAble() error {
	fmt.Println("One or more services use sudo. You may be prompted for your password.")
	var buffer bytes.Buffer

	buffer.WriteString("#!/bin/bash\n")
	buffer.WriteString("sudo echo Test > /dev/null\n")
	buffer.WriteString("ISCHILD=YES ")
	buffer.WriteString(strings.Join(os.Args, " "))
	buffer.WriteString("\n")

	logger.Printf("Writing sudoAbility script\n")
	file, err := createScriptFile("sudoAbility", buffer.String())
	if err != nil {
		return errors.WithStack(err)
	}

	logger.Printf("Launching sudoAbility script: %v\n", file.Name())
	err = syscall.Exec(file.Name(), []string{file.Name()}, os.Environ())
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func prepareForSudo() error {
	err := checkNotSudo()
	if err != nil {
		return errors.WithStack(err)
	}

	isChild := os.Getenv("ISCHILD")
	if isChild == "" {
		return errors.WithStack(ensureSudoAble())
	} else {
		logger.Println("Child process, sudo should be available")
	}
	return nil
}

func RemoveContents(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return errors.WithStack(err)
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return errors.WithStack(err)
	}
	for _, name := range names {
		err = os.RemoveAll(filepath.Join(dir, name))
		if err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

var flags = struct {
	config    string
	skipBuild bool
	watch     bool
	noPrompt  bool
	exclude   cli.StringSlice
}{}

func getOperationConfig() services.OperationConfig {
	return services.OperationConfig{
		Exclusions: []string(flags.exclude),
	}
}

type serviceOrGroupByName []services.ServiceOrGroup

func (s serviceOrGroupByName) Len() int {
	return len(s)
}
func (s serviceOrGroupByName) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s serviceOrGroupByName) Less(i, j int) bool {
	return len(s[i].GetName()) < len(s[j].GetName())
}

type ServiceConfigByPID []*services.ServiceConfig

func (s ServiceConfigByPID) Len() int {
	return len(s)
}
func (s ServiceConfigByPID) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s ServiceConfigByPID) Less(i, j int) bool {
	cmd1, _ := s[i].GetCommand()
	cmd2, _ := s[j].GetCommand()
	return cmd1.Pid < cmd2.Pid
}
