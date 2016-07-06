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
	"path"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"github.com/codegangsta/cli"
	"github.com/hpcloud/tail"
	"github.com/yext/errgo"
)

func main() {

	app := cli.NewApp()
	app.Name = "Edward"
	app.Usage = "Manage local microservices"
	app.Version = "1.0.0"
	app.Before = func(c *cli.Context) error {
		command := c.Args().First()
		if command == "start" || command == "stop" || command == "restart" {
			prepareForSudo()
		}

		err := EdwardConfig.initialize()
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

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

type EdwardConfiguration struct {
	Dir       string
	LogDir    string
	PidDir    string
	ScriptDir string
}

var EdwardConfig EdwardConfiguration = EdwardConfiguration{}

func createDirIfNeeded(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.MkdirAll(path, 0777)
	}
}

func (e *EdwardConfiguration) initialize() error {
	user, err := user.Current()
	if err != nil {
		return err
	}
	e.Dir = path.Join(user.HomeDir, ".edward")
	e.LogDir = path.Join(e.Dir, "logs")
	e.PidDir = path.Join(e.Dir, "pidFiles")
	e.ScriptDir = path.Join(e.Dir, "scriptFiles")
	createDirIfNeeded(e.Dir)
	createDirIfNeeded(e.LogDir)
	createDirIfNeeded(e.PidDir)
	createDirIfNeeded(e.ScriptDir)
	return nil
}

var groups map[string]*ServiceGroupConfig
var services map[string]*ServiceConfig

func thirdPartyService(name string, startCommand string, stopCommand string, started string) *ServiceConfig {
	pathStr := "$ALPHA"
	return &ServiceConfig{
		Name: name,
		Path: &pathStr,
		Env:  []string{"YEXT_RABBITMQ=localhost"},
		Commands: ServiceConfigCommands{
			Launch: startCommand,
			Stop:   stopCommand,
		},
		Properties: ServiceConfigProperties{
			Started: started,
		},
	}
}

func getAlpha() string {
	for _, env := range os.Environ() {
		pair := strings.Split(env, "=")
		if pair[0] == "ALPHA" {
			return pair[1]
		}
	}
	return ""
}

func addFoundServices() {
	foundServices, _, err := generateServices(getAlpha())
	if err != nil {
		log.Fatal(err)
	}

	for _, s := range foundServices {
		if _, found := services[s.Name]; !found {
			services[s.Name] = s
		}
	}
}

// getConfigPath identifies the location of edward.json, if any exists
func getConfigPath() string {

	if flags.config != "" {
		return flags.config
	}

	var pathOptions []string

	// Config file in Edward Config dir
	pathOptions = append(pathOptions, filepath.Join(EdwardConfig.Dir, "edward.json"))

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
			return path
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
	groups = make(map[string]*ServiceGroupConfig)
	services = make(map[string]*ServiceConfig)
}

func loadConfig() error {
	initEmptyConfig()

	configPath := getConfigPath()
	if configPath != "" {
		println("Loading configuration from", configPath)
		r, err := os.Open(configPath)
		if err != nil {
			return errgo.Mask(err)
		}
		config, err := LoadConfigWithDir(r, filepath.Dir(configPath))
		if err != nil {
			return errgo.Mask(err)
		}

		services = config.ServiceMap
		groups = config.GroupMap
		return nil
	} else {
		return errgo.New("No config file found")
	}

	return nil
}

func getServicesOrGroups(names []string) ([]ServiceOrGroup, error) {
	var outSG []ServiceOrGroup
	for _, name := range names {
		sg, err := getServiceOrGroup(name)
		if err != nil {
			return nil, err
		}
		outSG = append(outSG, sg)
	}
	return outSG, nil
}

func getServiceOrGroup(name string) (ServiceOrGroup, error) {
	if group, ok := groups[name]; ok {
		return group, nil
	}
	if service, ok := services[name]; ok {
		return service, nil
	}
	return nil, errors.New("Service or group not found")
}

func list(c *cli.Context) error {

	var groupNames []string
	var serviceNames []string
	for name, _ := range groups {
		groupNames = append(groupNames, name)
	}
	for name, _ := range services {
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

	var config Config
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
		config, err = LoadConfigWithDir(r, filepath.Dir(configPath))
		if err != nil {
			return errgo.Mask(err)
		}
	} else {
		config = EmptyConfig(filepath.Dir(configPath))
	}

	wd, err := os.Getwd()
	if err != nil {
		return errgo.Mask(err)
	}

	foundServices, _, err := generateServices(wd)
	if err != nil {
		return errgo.Mask(err)
	}
	err = config.AppendServices(foundServices)
	if err != nil {
		return errgo.Mask(err)
	}

	f, err := os.Create(configPath)
	if err != nil {
		return errgo.Mask(err)
	}

	defer f.Close()

	w := bufio.NewWriter(f)
	err = config.Save(w)
	if err != nil {
		return errgo.Mask(err)
	}

	println("Wrote to", configPath)

	return nil
}

func allStatus() {
	var statuses []ServiceStatus
	for _, service := range services {
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

func allServices() []ServiceOrGroup {
	var as []ServiceOrGroup
	for _, service := range services {
		as = append(as, service)
	}
	return as
}

func stop(c *cli.Context) error {
	var sgs []ServiceOrGroup
	var err error
	if len(c.Args()) == 0 {
		sgs = allServices()
	} else {
		sgs, err = getServicesOrGroups(c.Args())
		if err != nil {
			return err
		}
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
	if _, ok := groups[name]; ok {
		return errors.New("Cannot output group logs")
	}
	if service, ok := services[name]; ok {
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
	rebooted, err := hasRebooted(EdwardConfig.Dir)
	if err != nil {
		return errgo.Mask(err)
	}

	if rebooted {
		err = RemoveContents(EdwardConfig.PidDir)
		if err != nil {
			return errgo.Mask(err)
		}
		err = setRebootMarker(EdwardConfig.Dir)
		if err != nil {
			return errgo.Mask(err)
		}
	}

	return nil
}

var flags = struct {
	config string
}{}
