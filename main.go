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

	"github.com/codegangsta/cli"
	"github.com/hpcloud/tail"
	"github.com/yext/edward/config"
	"github.com/yext/edward/generators"
	"github.com/yext/edward/home"
	"github.com/yext/edward/reboot"
	"github.com/yext/edward/services"
	"github.com/yext/errgo"
)

var logger *log.Logger

func main() {

	logger = log.New(os.Stdout, "", 0)

	app := cli.NewApp()
	app.Name = "Edward"
	app.Usage = "Manage local microservices"
	app.Version = "1.1.0"
	app.Before = func(c *cli.Context) error {
		command := c.Args().First()

		err := home.EdwardConfig.Initialize()
		if err != nil {
			return errgo.Mask(err)
		}
		err = refreshForReboot()
		if err != nil {
			return errgo.Mask(err)
		}

		if command != "generate" {
			err = loadConfig()
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

	logger.Printf("%v v%v", app.Name, app.Version)

	err := app.Run(os.Args)
	if err != nil {
		logger.Fatal(err)
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

		serviceMap = cfg.ServiceMap
		groupMap = cfg.GroupMap
		return nil
	} else {
		return errgo.New("No config file found")
	}

	return nil
}

func sudoIfNeeded(sgs []services.ServiceOrGroup) {
	for _, sg := range sgs {
		if sg.IsSudo() {
			prepareForSudo()
		}
	}
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
	for name, _ := range groupMap {
		groupNames = append(groupNames, name)
	}
	for name, _ := range serviceMap {
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
		cfg = config.EmptyConfig(filepath.Dir(configPath))
	}

	wd, err := os.Getwd()
	if err != nil {
		return errgo.Mask(err)
	}

	foundServices, err := generators.GenerateServices(wd)
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

	println("Wrote to", configPath)

	return nil
}

func allStatus() {
	var statuses []services.ServiceStatus
	for _, service := range serviceMap {
		statuses = append(statuses, service.GetStatus()...)
	}
	for _, status := range statuses {
		if status.Status != "STOPPED" {
			println(status.Service.Name, ":", status.Status)
		}
	}
}

func status(c *cli.Context) error {

	if len(c.Args()) == 0 {
		allStatus()
		return nil
	}

	sgs, err := getServicesOrGroups(c.Args())
	if err != nil {
		return err
	}
	for _, s := range sgs {
		statuses := s.GetStatus()
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
	sudoIfNeeded(sgs)

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
	sudoIfNeeded(sgs)
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
	sudoIfNeeded(sgs)
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
		command := service.GetCommand()
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

func checkNotSudo() {
	user, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	if user.Uid == "0" {
		log.Fatal("edward should not be run with sudo")
	}
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

func ensureSudoAble() {
	var buffer bytes.Buffer

	buffer.WriteString("#!/bin/bash\n")
	buffer.WriteString("sudo echo Test > /dev/null\n")
	buffer.WriteString("ISCHILD=YES ")
	buffer.WriteString(strings.Join(os.Args, " "))
	buffer.WriteString("\n")

	file, err := createScriptFile("sudoAbility", buffer.String())
	if err != nil {
		log.Fatal(err)
	}

	err = syscall.Exec(file.Name(), []string{file.Name()}, os.Environ())
	if err != nil {
		log.Fatal(err)
	}
}

func prepareForSudo() {
	checkNotSudo()

	isChild := os.Getenv("ISCHILD")
	if isChild == "" {
		ensureSudoAble()
		return
	}
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

func refreshForReboot() error {
	if os.Getenv("EDWARD_NO_REBOOT") == "1" {
		fmt.Println("Reboot detection disabled")
		return nil
	}

	rebooted, err := reboot.HasRebooted(home.EdwardConfig.Dir)
	if err != nil {
		return errgo.Mask(err)
	}

	if rebooted {
		err = RemoveContents(home.EdwardConfig.PidDir)
		if err != nil {
			return errgo.Mask(err)
		}
		err = reboot.SetRebootMarker(home.EdwardConfig.Dir)
		if err != nil {
			return errgo.Mask(err)
		}
	}

	return nil
}

var flags = struct {
	config string
}{}
