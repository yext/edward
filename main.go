package main

import (
	"errors"
	"log"
	"os"

	"github.com/codegangsta/cli"
)

var groups map[string]*ServiceGroupConfig
var services map[string]*ServiceConfig

func thirdPartyService(name string, command string) *ServiceConfig {
	pathStr := "$ALPHA"
	return &ServiceConfig{
		Name: name,
		Path: &pathStr,
		Commands: struct {
			Build  string
			Launch string
		}{
			Build:  "",
			Launch: command,
		},
	}
}

func playService(name string) *ServiceConfig {
	pathStr := "$ALPHA"
	return &ServiceConfig{
		Name: name,
		Path: &pathStr,
		Commands: struct {
			Build  string
			Launch string
		}{
			Build:  "python tools/icbm/build.py :" + name + "_dev",
			Launch: "YEXT_RABBITMQ=localhost thirdparty/play/play test src/com/yext/" + name,
		},
		Properties: struct {
			Started string
			Custom  map[string]string
		}{
			Started: "started",
		},
	}
}

func javaService(name string) *ServiceConfig {
	pathStr := "$ALPHA"
	return &ServiceConfig{
		Name: name,
		Path: &pathStr,
		Commands: struct {
			Build  string
			Launch string
		}{
			Build:  "python tools/icbm/build.py :" + name,
			Launch: "YEXT_RABBITMQ=localhost src/com/yext/" + name,
		},
		Properties: struct {
			Started string
			Custom  map[string]string
		}{
			Started: "started",
		},
	}
}

func goService(name string, goPackage string) *ServiceConfig {
	pathStr := "$ALPHA"
	return &ServiceConfig{
		Name: name,
		Path: &pathStr,
		Commands: struct {
			Build  string
			Launch string
		}{
			Build:  "go install " + goPackage,
			Launch: "YEXT_RABBITMQ=localhost " + name,
		},
		Properties: struct {
			Started string
			Custom  map[string]string
		}{
			Started: "started",
		},
	}
}

func loadConfig() {
	// TODO: Load configuration from the config file and populate the service and groups variables

	groups = make(map[string]*ServiceGroupConfig)
	services = make(map[string]*ServiceConfig)

	services["rabbitmq"] = thirdPartyService("rabbitmq", "sudo rabbitmq-server")
	// TODO: haproxy actually needs a kill -9 to effectively die
	services["haproxy"] = thirdPartyService("haproxy", "sudo $ALPHA/tools/bin/haproxy_localhost.sh")

	services["admin2"] = playService("admin2")
	services["users"] = playService("users")
	services["storm"] = playService("storm")
	services["ProfileServer"] = javaService("ProfileServer")

	services["sites-staging"] = goService("sites-staging", "yext/pages/sites/sites-staging")
	services["sites-storm"] = goService("sites-storm", "yext/pages/sites/sites-storm")
	services["sites-cog"] = goService("sites-cog", "yext/pages/sites/sites-cog")

	groups["thirdparty"] = &ServiceGroupConfig{
		Name: "thirdparty",
		Services: []*ServiceConfig{
			services["rabbitmq"],
			services["haproxy"],
		},
	}

	groups["base"] = &ServiceGroupConfig{
		Name: "base",
		Groups: []*ServiceGroupConfig{
			groups["thirdparty"],
		},
		Services: []*ServiceConfig{
			services["admin2"],
			services["users"],
			services["storm"],
			services["ProfileServer"],
		},
	}

	groups["pages"] = &ServiceGroupConfig{
		Name: "pages",
		Groups: []*ServiceGroupConfig{
			groups["base"],
		},
		Services: []*ServiceConfig{
			services["sites-staging"],
			services["sites-storm"],
			services["sites-cog"],
		},
	}
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
	println("Services and groups")
	println("Groups:")
	for name, _ := range groups {
		println("\t", name)
	}
	println("Services:")
	for name, _ := range services {
		println("\t", name)
	}
}

func generate(c *cli.Context) {
	println("Generate config")
}

func status(c *cli.Context) {
	println("Status")
}

func messages(c *cli.Context) {
	println("Messages")
}

func start(c *cli.Context) {
	name := c.Args()[0]
	s, err := getServiceOrGroup(name)
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

func stop(c *cli.Context) {
	name := c.Args()[0]
	s, err := getServiceOrGroup(name)
	if err != nil {
		log.Fatal(err)
	}
	err = s.Stop()
	if err != nil {
		log.Fatal(err)
	}
}

func restart(c *cli.Context) {
	name := c.Args()[0]
	s, err := getServiceOrGroup(name)
	if err != nil {
		log.Fatal(err)
	}
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

func doLog(c *cli.Context) {
	println("Log")
}

func main() {

	app := cli.NewApp()
	app.Name = "Edward"
	app.Usage = "Manage local microservices"
	app.Before = func(c *cli.Context) error {
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
