package main

import (
	"os"

	"github.com/codegangsta/cli"
)

var groups []ServiceGroupConfig
var services []ServiceConfig

type ServiceConfigFile struct {
	Services []ServiceConfig
	Groups   []ServiceGroupConfig
}

// ServiceGroupConfig is a group of services that can be managed together
type ServiceGroupConfig struct {
	// A name for this group, used to identify it in commands
	Name string
	// Paths to child service config files
	ServicePaths []string
	// Full services contained within this group
	Services []*ServiceConfig
}

// ServiceConfig represents a service that can be managed by Edward
type ServiceConfig struct {
	// Service name, used to identify in commands
	Name string
	// Optional path to service. If nil, uses cwd
	Path *string
	// Commands for managing the service
	Commands struct {
		// Command to build
		Build string
		// Command to launch
		Launch string
	}
	// Service state properties that can be obtained from logs
	Properties struct {
		// Regex to detect a line indicating the service has started successfully
		Started string
		// Custom properties, mapping a property name to a regex
		Custom map[string]string
	}
}

func loadConfig() {
	// TODO: Load configuration from the config file and populate the service and groups variables
}

func list(c *cli.Context) {
	println("List services")
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
	println("Start")
}

func stop(c *cli.Context) {
	println("Stop")
}

func restart(c *cli.Context) {
	println("Restart")
}

func log(c *cli.Context) {
	println("Log")
}

func main() {

	app := cli.NewApp()
	app.Name = "Edward"
	app.Usage = "Manage local microservices"
	app.Before = func(c *cli.Context) error {
		loadConfig()
		println("Before")
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
			Action: log,
		},
	}

	app.Run(os.Args)
}
