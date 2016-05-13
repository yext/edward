package main

import (
	"bytes"
	"errors"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"github.com/codegangsta/cli"
	"github.com/hpcloud/tail"
)

type EdwardConfiguration struct {
	Dir    string
	LogDir string
	PidDir string
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
	createDirIfNeeded(e.Dir)
	createDirIfNeeded(e.LogDir)
	createDirIfNeeded(e.PidDir)
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

func getConfigPath() string {
	return filepath.Join(EdwardConfig.Dir, "edward.json")
}

func loadConfig() {
	groups = make(map[string]*ServiceGroupConfig)
	services = make(map[string]*ServiceConfig)

	configPath := getConfigPath()

	if _, err := os.Stat(configPath); err == nil {
		println("Loading configuration from", configPath)
		r, err := os.Open(configPath)
		if err != nil {
			log.Fatal(err)
		}
		config, err := LoadConfig(r)
		if err != nil {
			log.Fatal(err)
		}

		services = config.ServiceMap
		groups = config.GroupMap
		return
	} else {
		addFoundServices()
		applyHardCodedServicesAndGroups()
	}

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

func list(c *cli.Context) {

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
}

func generate(c *cli.Context) {

	// Add any new services to the config as appropriate
	addFoundServices()

	configPath := getConfigPath()

	if err := generateConfigFile(configPath); err != nil {
		log.Fatal(err)
	}
	println("Wrote to", configPath)

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

func status(c *cli.Context) {

	if len(c.Args()) == 0 {
		allStatus()
		return
	}

	sgs, err := getServicesOrGroups(c.Args())
	if err != nil {
		log.Fatal(err)
	}
	for _, s := range sgs {
		statuses := s.GetStatus()
		for _, status := range statuses {
			println(status.Service.Name, ":", status.Status)
		}
	}
}

func messages(c *cli.Context) {
	log.Fatal("Unimplemented")
}

func start(c *cli.Context) {
	sgs, err := getServicesOrGroups(c.Args())
	if err != nil {
		log.Fatal(err)
	}
	for _, s := range sgs {
		println("==== Build Phase ====")
		err = s.Build()
		if err != nil {
			log.Fatal("Error building ", s.GetName(), ": ", err)
		}
		println("==== Launch Phase ====")
		err = s.Start()
		if err != nil {
			log.Fatal("Error launching ", s.GetName(), ": ", err)
		}
	}
}

func stop(c *cli.Context) {
	sgs, err := getServicesOrGroups(c.Args())
	if err != nil {
		log.Fatal(err)
	}
	for _, s := range sgs {
		err = s.Stop()
		if err != nil {
			log.Fatal(err)
		}
	}
}

func restart(c *cli.Context) {
	sgs, err := getServicesOrGroups(c.Args())
	if err != nil {
		log.Fatal(err)
	}
	for _, s := range sgs {
		err = s.Stop()
		if err != nil {
			log.Fatal(err)
		}
		err = s.Build()
		if err != nil {
			log.Fatal(err)
		}
		err = s.Start()
		if err != nil {
			log.Fatal(err)
		}
	}
}

func doLog(c *cli.Context) {
	if len(c.Args()) > 1 {
		log.Fatal(errors.New("Cannot output multiple service logs"))
	}
	name := c.Args()[0]
	if _, ok := groups[name]; ok {
		log.Fatal(errors.New("Cannot output group logs"))
	}
	if service, ok := services[name]; ok {
		command := service.GetCommand()
		runLog := command.Logs.Run
		t, err := tail.TailFile(runLog, tail.Config{Follow: true})
		if err != nil {
			log.Fatal(err)
		}
		for line := range t.Lines {
			println(line.Text)
		}
		return
	}
	log.Fatal("Service not found:", name)
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

func main() {

	checkNotSudo()

	isChild := os.Getenv("ISCHILD")
	if isChild == "" {
		ensureSudoAble()
		return
	}

	app := cli.NewApp()
	app.Name = "Edward"
	app.Usage = "Manage local microservices"
	app.Before = func(c *cli.Context) error {
		err := EdwardConfig.initialize()
		if err != nil {
			return err
		}
		loadConfig()
		return nil
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
			Name:   "log",
			Usage:  "Tail the log for a service",
			Action: doLog,
		},
	}

	app.Run(os.Args)
}
