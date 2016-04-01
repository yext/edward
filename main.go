package main

import (
	"errors"
	"log"
	"os"
	"os/user"
	"path"

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
		Commands: ServiceConfigCommands{
			Launch: startCommand,
			Stop:   stopCommand,
		},
		Properties: ServiceConfigProperties{
			Started: started,
		},
	}
}

func playService(name string) *ServiceConfig {
	pathStr := "$ALPHA"
	return &ServiceConfig{
		Name: name,
		Path: &pathStr,
		Commands: ServiceConfigCommands{
			Build:  "python tools/icbm/build.py :" + name + "_dev",
			Launch: "YEXT_RABBITMQ=localhost thirdparty/play/play test src/com/yext/" + name,
		},
		Properties: ServiceConfigProperties{
			Started: "Server is up and running",
		},
	}
}

func javaService(name string) *ServiceConfig {
	pathStr := "$ALPHA"
	return &ServiceConfig{
		Name: name,
		Path: &pathStr,
		Commands: ServiceConfigCommands{
			Build:  "python tools/icbm/build.py :" + name,
			Launch: "YEXT_RABBITMQ=localhost YEXT_SITE=office JVM_ARGS='-Xmx3G' build/" + name + "/" + name,
		},
		Properties: ServiceConfigProperties{
			Started: "Listening",
		},
	}
}

func goService(name string, goPackage string) *ServiceConfig {
	pathStr := "$ALPHA"
	return &ServiceConfig{
		Name: name,
		Path: &pathStr,
		Commands: ServiceConfigCommands{
			Build:  "go install " + goPackage,
			Launch: "YEXT_RABBITMQ=localhost " + name,
		},
		Properties: ServiceConfigProperties{
			Started: "Listening",
		},
	}
}

func loadConfig() {
	// TODO: Load configuration from the config file and populate the service and groups variables

	groups = make(map[string]*ServiceGroupConfig)
	services = make(map[string]*ServiceConfig)

	services["rabbitmq"] = thirdPartyService("rabbitmq", "sudo rabbitmq-server", "sudo rabbitmqctl stop", "completed")
	// TODO: haproxy actually needs a kill -9 to effectively die
	// TODO: haproxy also doesn't have an effective start output
	services["haproxy"] = thirdPartyService("haproxy", "sudo $ALPHA/tools/bin/haproxy_localhost.sh", "", "backend")

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

func main() {

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
