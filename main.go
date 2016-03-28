package main

import (
	"log"
	"os"

	"github.com/codegangsta/cli"
)

var groups map[string]*ServiceGroupConfig
var services map[string]*ServiceConfig

func loadConfig() {
	// TODO: Load configuration from the config file and populate the service and groups variables

	groups = make(map[string]*ServiceGroupConfig)
	services = make(map[string]*ServiceConfig)

	pathStr := "$ALPHA"

	services["admin2"] = &ServiceConfig{
		Name: "admin2",
		Path: &pathStr,
		Commands: struct {
			Build  string
			Launch string
		}{
			Build:  "python tools/icbm/build.py :admin2_dev",
			Launch: "thirdparty/play/play test src/com/yext/admin2",
		},
		Properties: struct {
			Started string
			Custom  map[string]string
		}{
			Started: "started",
		},
	}
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
	if _, ok := groups[name]; ok {
		println("Will start all services in this group")
		// TODO: Iterate over services and start
		return
	}
	if val, ok := services[name]; ok {
		println("Starting service", name)
		err := val.Start()
		if err != nil {
			log.Fatal(err)
		}
		return
	}
	println("Unknown group or service", name)
}

func stop(c *cli.Context) {

	name := c.Args()[0]
	if _, ok := groups[name]; ok {
		println("Will stop all services in this group")
		// TODO: Iterate over services and start
		return
	}
	if val, ok := services[name]; ok {
		println("Stopping service", name)
		err := val.Stop()
		if err != nil {
			log.Fatal(err)
		}
		return
	}
	println("Unknown group or service", name)
}

func restart(c *cli.Context) {
	println("Restart")
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
