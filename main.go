package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/codegangsta/cli"
	"github.com/hashicorp/go-version"
	"github.com/hpcloud/tail"
	"github.com/yext/edward/config"
	"github.com/yext/edward/generators"
	"github.com/yext/edward/home"
	"github.com/yext/edward/services"
	"github.com/yext/edward/updates"
	"github.com/yext/errgo"
)

var logger *log.Logger

const edwardVersion = "1.4.0"

func main() {

	if err := home.EdwardConfig.Initialize(); err != nil {
		log.Fatal(err)
	}

	logger = log.New(&lumberjack.Logger{
		Filename:   filepath.Join(home.EdwardConfig.EdwardLogDir, "edward.log"),
		MaxSize:    500, // megabytes
		MaxBackups: 3,
		MaxAge:     365, //days
	}, "", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)

	app := cli.NewApp()
	app.Name = "Edward"
	app.Usage = "Manage local microservices"
	app.Version = edwardVersion
	app.Before = func(c *cli.Context) error {
		command := c.Args().First()

		if command != "generate" {
			err := loadConfig()
			if err != nil {
				return errgo.Mask(err)
			}
		} else {
			initEmptyConfig()
		}

		return nil
	}
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "config, c",
			Usage:       "Use service configuration file at `PATH`",
			Destination: &(flags.config),
		},
	}
	app.Commands = []cli.Command{
		{
			Name:   "list",
			Usage:  "List available services",
			Action: list,
		},
		{
			Name:   "generate",
			Usage:  "Generate Edward config for a source tree",
			Action: generate,
		},
		{
			Name:   "status",
			Usage:  "Display service status",
			Action: status,
		},
		{
			Name:   "messages",
			Usage:  "Show messages from services",
			Action: messages,
		},
		{
			Name:   "start",
			Usage:  "Build and launch a service",
			Action: start,
		},
		{
			Name:   "stop",
			Usage:  "Stop a service",
			Action: stop,
		},
		{
			Name:   "restart",
			Usage:  "Rebuild and relaunch a service",
			Action: restart,
		},
		{
			Name:    "log",
			Aliases: []string{"tail"},
			Usage:   "Tail the log for a service",
			Action:  doLog,
		},
	}

	logger.Printf("=== %v v%v ===\n", app.Name, app.Version)
	logger.Printf("Args: %v\n", os.Args)
	defer logger.Printf("=== Exiting ===\n")

	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err)
		logger.Println(err)
	}

	updateAvailable, latestVersion, err := updates.UpdateAvailable("github.com/yext/edward", edwardVersion, filepath.Join(home.EdwardConfig.Dir, ".updatecache"), logger)
	if err != nil {
		fmt.Println(err)
		logger.Println(err)
		return
	}
	if updateAvailable {
		fmt.Printf("A new version of Edward is available (%v), update with:\n\tgo get -u github.com/yext/edward\n", latestVersion)
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

	// Config file at root of working dir's git repo, if any
	gitRoot, err := gitRoot()
	if err == nil {
		pathOptions = append(pathOptions, filepath.Join(gitRoot, "edward.json"))

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
			return errgo.Mask(err)
		}
		cfg, err := config.LoadConfigWithDir(r, filepath.Dir(configPath), logger)
		if err != nil {
			return errgo.Mask(err)
		}

		if cfg.MinEdwardVersion != "" {
			// Check that this config is supported by this version
			minVersion, err1 := version.NewVersion(cfg.MinEdwardVersion)
			if err1 != nil {
				return errgo.Mask(err)
			}
			currentVersion, err2 := version.NewVersion(edwardVersion)
			if err2 != nil {
				return errgo.Mask(err)
			}
			if currentVersion.LessThan(minVersion) {
				return errgo.New("this config requires at least version " + cfg.MinEdwardVersion)
			}
		}

		serviceMap = cfg.ServiceMap
		groupMap = cfg.GroupMap
		return nil
	}

	return errgo.New("No config file found")

}

func sudoIfNeeded(sgs []services.ServiceOrGroup) error {
	for _, sg := range sgs {
		if sg.IsSudo() {
			logger.Printf("sudo required for %v\n", sg.GetName())
			return errgo.Mask(prepareForSudo())
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

func list(c *cli.Context) error {

	var groupNames []string
	var serviceNames []string
	for name := range groupMap {
		groupNames = append(groupNames, name)
	}
	for name := range serviceMap {
		serviceNames = append(serviceNames, name)
	}

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
			return errgo.Mask(err)
		}
		cfg, err = config.LoadConfigWithDir(r, filepath.Dir(configPath), logger)
		if err != nil {
			return errgo.Mask(err)
		}
	} else {
		cfg = config.EmptyConfig(filepath.Dir(configPath), logger)
	}

	wd, err := os.Getwd()
	if err != nil {
		return errgo.Mask(err)
	}

	foundServices, err := generators.GenerateServices(wd, c.Args())
	if err != nil {
		return errgo.Mask(err)
	}
	foundServices, err = cfg.NormalizeServicePaths(wd, foundServices)
	if err != nil {
		return errgo.Mask(err)
	}
	err = cfg.AppendServices(foundServices)
	if err != nil {
		return errgo.Mask(err)
	}

	f, err := os.Create(configPath)
	if err != nil {
		return errgo.Mask(err)
	}

	defer f.Close()

	w := bufio.NewWriter(f)
	err = cfg.Save(w)
	if err != nil {
		return errgo.Mask(err)
	}
	err = w.Flush()
	if err != nil {
		return errgo.Mask(err)
	}

	println("Wrote to", configPath)

	return nil
}

func allStatus() error {
	var statuses []services.ServiceStatus
	for _, service := range serviceMap {
		s, err := service.Status()
		if err != nil {
			return errgo.Mask(err)
		}
		statuses = append(statuses, s...)
	}
	for _, status := range statuses {
		if status.Status != "STOPPED" {
			println(status.Service.Name, ":", status.Status)
		}
	}
	return nil
}

func status(c *cli.Context) error {

	if len(c.Args()) == 0 {
		return errgo.Mask(allStatus())
	}

	sgs, err := getServicesOrGroups(c.Args())
	if err != nil {
		return err
	}
	for _, s := range sgs {
		statuses, err := s.Status()
		if err != nil {
			return errgo.Mask(err)
		}
		for _, status := range statuses {
			println(status.Service.Name, ":", status.Status)
		}
	}
	return nil
}

func messages(c *cli.Context) error {
	return errors.New("Unimplemented")
}

func start(c *cli.Context) error {
	sgs, err := getServicesOrGroups(c.Args())
	if err != nil {
		return err
	}
	err = sudoIfNeeded(sgs)
	if err != nil {
		return errgo.Mask(err)
	}

	for _, s := range sgs {
		println("==== Build Phase ====")
		err = s.Build()
		if err != nil {
			return errors.New("Error building " + s.GetName() + ": " + err.Error())
		}
		println("==== Launch Phase ====")
		err = s.Start()
		if err != nil {
			return errors.New("Error launching " + s.GetName() + ": " + err.Error())
		}
	}
	return nil
}

func allServices() []services.ServiceOrGroup {
	var as []services.ServiceOrGroup
	for _, service := range serviceMap {
		as = append(as, service)
	}
	return as
}

func stop(c *cli.Context) error {
	var sgs []services.ServiceOrGroup
	var err error
	if len(c.Args()) == 0 {
		sgs = allServices()
	} else {
		sgs, err = getServicesOrGroups(c.Args())
		if err != nil {
			return err
		}
	}
	err = sudoIfNeeded(sgs)
	if err != nil {
		return errgo.Mask(err)
	}

	for _, s := range sgs {
		_ = s.Stop()
	}
	return nil
}

func restart(c *cli.Context) error {
	sgs, err := getServicesOrGroups(c.Args())
	if err != nil {
		return err
	}
	err = sudoIfNeeded(sgs)
	if err != nil {
		return errgo.Mask(err)
	}
	for _, s := range sgs {
		_ = s.Stop()
		err = s.Build()
		if err != nil {
			return err
		}
		err = s.Start()
		if err != nil {
			return err
		}
	}
	return nil
}

func doLog(c *cli.Context) error {
	if len(c.Args()) > 1 {
		return errors.New("Cannot output multiple service logs")
	}
	name := c.Args()[0]
	if _, ok := groupMap[name]; ok {
		return errors.New("Cannot output group logs")
	}
	if service, ok := serviceMap[name]; ok {
		command, err := service.GetCommand()
		if err != nil {
			return errgo.Mask(err)
		}
		runLog := command.Logs.Run
		t, err := tail.TailFile(runLog, tail.Config{Follow: true})
		if err != nil {
			return nil
		}
		for line := range t.Lines {
			println(line.Text)
		}
		return nil
	}
	return errors.New("Service not found: " + name)
}

func checkNotSudo() error {
	user, err := user.Current()
	if err != nil {
		return errgo.Mask(err)
	}
	if user.Uid == "0" {
		return errgo.New("edward should not be fun with sudo")
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
	var buffer bytes.Buffer

	buffer.WriteString("#!/bin/bash\n")
	buffer.WriteString("sudo echo Test > /dev/null\n")
	buffer.WriteString("ISCHILD=YES ")
	buffer.WriteString(strings.Join(os.Args, " "))
	buffer.WriteString("\n")

	logger.Printf("Writing sudoAbility script\n")
	file, err := createScriptFile("sudoAbility", buffer.String())
	if err != nil {
		return errgo.Mask(err)
	}

	logger.Printf("Launching sudoAbility script: %v\n", file.Name())
	err = syscall.Exec(file.Name(), []string{file.Name()}, os.Environ())
	if err != nil {
		return errgo.Mask(err)
	}
	return nil
}

func prepareForSudo() error {
	err := checkNotSudo()
	if err != nil {
		return errgo.Mask(err)
	}

	isChild := os.Getenv("ISCHILD")
	if isChild == "" {
		return errgo.Mask(ensureSudoAble())
	} else {
		logger.Println("Child process, sudo should be available")
	}
	return nil
}

func RemoveContents(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, name := range names {
		err = os.RemoveAll(filepath.Join(dir, name))
		if err != nil {
			return err
		}
	}
	return nil
}

var flags = struct {
	config string
}{}
