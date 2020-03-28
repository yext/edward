package edward

import (
	"log"
	"os"

	"github.com/pkg/errors"
	"github.com/yext/edward/builder"
	"github.com/yext/edward/home"
	"github.com/yext/edward/instance"
	"github.com/yext/edward/output"
	"github.com/yext/edward/services"
	"github.com/yext/edward/tracker"
	"github.com/yext/edward/ui"
	"github.com/yext/edward/ui/terminal"
	"github.com/yext/edward/worker"
)

type Client struct {
	UI ui.Provider

	Config string

	ServiceChecks func([]services.ServiceOrGroup) error

	EdwardExecutable string

	LogFile string // Path to log file

	Follower TaskFollower

	// Prevent build, launch and stop phases from running concurrently
	DisableConcurrentPhases bool

	WorkingDir string

	telemetryScript string

	basePath   string
	groupMap   map[string]*services.ServiceGroupConfig
	serviceMap map[string]*services.ServiceConfig

	Tags []string // Tags to distinguish runners started by this instance of edward

	DirConfig *home.EdwardConfiguration

	Backends map[string]string // Service overrides for backends
}

type TaskFollower interface {
	Handle(update tracker.Task)
	Done()
}

// NewClient creates an edward client an empty configuration
func NewClient() (*Client, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	dirCfg, err := home.NewConfiguration("")
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &Client{
		UI:         &terminal.Provider{},
		Follower:   output.NewFollower(),
		WorkingDir: wd,
		groupMap:   make(map[string]*services.ServiceGroupConfig),
		serviceMap: make(map[string]*services.ServiceConfig),
		DirConfig:  dirCfg,
	}, nil
}

// NewClientWithConfig creates an Edward client and loads the config from the given path
func NewClientWithConfig(configPath, version string) (*Client, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	dirCfg, err := home.NewConfiguration("")
	if err != nil {
		return nil, errors.WithStack(err)
	}

	client := &Client{
		UI:         &terminal.Provider{},
		Follower:   output.NewFollower(),
		WorkingDir: wd,
		Config:     configPath,
		groupMap:   make(map[string]*services.ServiceGroupConfig),
		serviceMap: make(map[string]*services.ServiceConfig),
		DirConfig:  dirCfg,
	}
	err = client.LoadConfig(version)
	return client, errors.WithStack(err)
}

func (c *Client) BasePath() string {
	return c.basePath
}

func (c *Client) ServiceMap() map[string]*services.ServiceConfig {
	return c.serviceMap
}

func (c *Client) startAndTrack(sgs []services.ServiceOrGroup, skipBuild bool, noWatch bool, exclude []string, edwardExecutable string) error {
	cfg := services.OperationConfig{
		WorkingDir:       c.WorkingDir,
		EdwardExecutable: edwardExecutable,
		Exclusions:       exclude,
		SkipBuild:        skipBuild,
		NoWatch:          noWatch,
		Tags:             c.Tags,
		LogFile:          c.LogFile,
		Backends:         c.Backends,
	}

	task := tracker.NewTask(c.Follower.Handle)
	defer c.Follower.Done()

	poolSize := 1
	if c.DisableConcurrentPhases {
		poolSize = 0
	}
	p := worker.NewPool(poolSize)
	p.Start()
	var err error
	err = services.DoForServices(sgs, task, func(s *services.ServiceConfig, overrides services.ContextOverride, task tracker.Task) error {
		if skipBuild {
			log.Println("skipping build phase")
			err = instance.Launch(c.DirConfig, s, cfg, overrides, task, p)
			if err != nil {
				return errors.WithMessage(err, "Error launching "+s.GetName())
			}
		} else {
			if cfg.IsExcluded(s) {
				return nil
			}
			log.Printf("Building: %s", s.Name)
			b := builder.New(cfg, overrides)
			err := b.Build(c.DirConfig, task, s)
			if err != nil {
				return errors.WithMessage(err, "Error starting "+s.GetName()+": build")
			}
			log.Printf("Launching: %s", s.Name)
			err = instance.Launch(c.DirConfig, s, cfg, overrides, task, p)
			return errors.WithMessage(err, "Error starting "+s.GetName()+": launch")
		}
		return nil
	})
	if err != nil {
		return errors.WithStack(err)
	}

	p.Stop()
	_ = <-p.Complete()
	return p.Err()
}

func (c *Client) askForConfirmation(question string) bool {

	// Skip confirmations for children. For example, for enabling sudo.
	isChild := os.Getenv("ISCHILD")
	if isChild != "" {
		return true
	}

	return c.UI.Confirm(question)
}
