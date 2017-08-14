package edward

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/pkg/errors"
	"github.com/yext/edward/common"
	"github.com/yext/edward/config"
	"github.com/yext/edward/generators"
	"github.com/yext/edward/output"
	"github.com/yext/edward/runner"
	"github.com/yext/edward/services"
	"github.com/yext/edward/tracker"
	"github.com/yext/edward/worker"
)

type Client struct {
	Logger *log.Logger

	Output io.Writer

	Config string
	Force  bool

	ServiceChecks func([]services.ServiceOrGroup) error

	EdwardExecutable string

	Follower TaskFollower
}

type TaskFollower interface {
	Handle(update tracker.Task)
	Done()
}

func NewClient() *Client {
	return &Client{
		Output:   os.Stdout,
		Follower: output.NewFollower(),
	}
}

func (c *Client) Version() string {
	return common.EdwardVersion
}

func (c *Client) Start(names []string, skipBuild bool, tail bool, noWatch bool, exclude []string) error {
	if len(names) == 0 {
		return errors.New("At least one service or group must be specified")
	}

	sgs, err := config.GetServicesOrGroups(names)
	if err != nil {
		return errors.WithStack(err)
	}
	if c.ServiceChecks != nil {
		err = c.ServiceChecks(sgs)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	err = c.startAndTrack(sgs, skipBuild, tail, noWatch, exclude, c.EdwardExecutable)
	if err != nil {
		return errors.WithStack(err)
	}
	if tail {
		return errors.WithStack(c.tailFromFlag(names))
	}

	return nil
}

func (c *Client) Restart(names []string, force bool, skipBuild bool, tail bool, noWatch bool, exclude []string) error {

	if len(names) == 0 {
		// Prompt user to confirm the restart
		if !force && !askForConfirmation("Are you sure you want to restart?") {
			return nil
		}
		c.restartAll(skipBuild, tail, noWatch, exclude)
	} else {
		err := c.restartOneOrMoreServices(names, skipBuild, tail, noWatch, exclude)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	if tail {
		return errors.WithStack(c.tailFromFlag(names))
	}
	return nil
}

func (c *Client) restartAll(skipBuild bool, tail bool, noWatch bool, exclude []string) error {
	var as []*services.ServiceConfig
	for _, service := range config.GetServiceMap() {
		s, err := service.Status()
		if err != nil {
			return errors.WithStack(err)
		}
		for _, status := range s {
			if status.Status != services.StatusStopped {
				as = append(as, service)
			}
		}
	}

	sort.Sort(serviceConfigByPID(as))
	var serviceNames []string
	for _, service := range as {
		serviceNames = append(serviceNames, service.Name)
	}

	return errors.WithStack(c.restartOneOrMoreServices(serviceNames, skipBuild, tail, noWatch, exclude))
}

func (c *Client) restartOneOrMoreServices(serviceNames []string, skipBuild bool, tail bool, noWatch bool, exclude []string) error {
	sgs, err := config.GetServicesOrGroups(serviceNames)
	if err != nil {
		return errors.WithStack(err)
	}
	if c.ServiceChecks != nil {
		if err = c.ServiceChecks(sgs); err != nil {
			return errors.WithStack(err)
		}
	}

	cfg := services.OperationConfig{
		EdwardExecutable: c.EdwardExecutable,
		Exclusions:       exclude,
		SkipBuild:        skipBuild,
		NoWatch:          noWatch,
	}

	task := tracker.NewTask(c.Follower.Handle)
	defer c.Follower.Done()

	launchPool := worker.NewPool(1)
	launchPool.Start()
	defer func() {
		launchPool.Stop()
		_ = <-launchPool.Complete()
	}()
	for _, s := range sgs {
		err = s.Restart(cfg, services.ContextOverride{}, task, launchPool)
		if err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

func (c *Client) Stop(names []string, exclude []string) error {
	var sgs []services.ServiceOrGroup
	var err error
	if len(names) == 0 {
		allSrv := config.GetAllServicesSorted()
		for _, service := range allSrv {
			var s []services.ServiceStatus
			s, err = service.Status()
			if err != nil {
				return errors.WithStack(err)
			}
			for _, status := range s {
				if status.Status != services.StatusStopped {
					sgs = append(sgs, service)
				}
			}
		}
	} else {
		sgs, err = config.GetServicesOrGroups(names)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	// Perform required checks and actions for services
	if c.ServiceChecks != nil {
		if err = c.ServiceChecks(sgs); err != nil {
			return errors.WithStack(err)
		}
	}

	cfg := services.OperationConfig{
		EdwardExecutable: c.EdwardExecutable,
		Exclusions:       exclude,
	}

	task := tracker.NewTask(c.Follower.Handle)
	defer c.Follower.Done()

	p := worker.NewPool(3)
	p.Start()
	defer func() {
		p.Stop()
		_ = <-p.Complete()
	}()
	for _, s := range sgs {
		_ = s.Stop(cfg, services.ContextOverride{}, task, p)
	}
	return nil
}

func (c *Client) Status(names []string) error {
	var sgs []services.ServiceOrGroup
	var err error
	if len(names) == 0 {
		for _, service := range config.GetAllServicesSorted() {
			var s []services.ServiceStatus
			s, err = service.Status()
			if err != nil {
				return errors.WithStack(err)
			}
			for _, status := range s {
				if status.Status != services.StatusStopped {
					sgs = append(sgs, service)
				}
			}
		}
		if len(sgs) == 0 {
			fmt.Println("No services are running")
			return nil
		}
	} else {

		sgs, err = config.GetServicesOrGroups(names)
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
		"Stdout",
		"Stderr",
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
				strconv.Itoa(status.StdoutCount) + " lines",
				strconv.Itoa(status.StderrCount) + " lines",
				status.StartTime.Format("2006-01-02 15:04:05"),
			})
		}
	}
	table.Render()
	return nil
}

func (c *Client) List() error {
	groups := config.GetAllGroupsSorted()
	services := config.GetAllServicesSorted()

	fmt.Fprintln(c.Output, "Services and groups")
	fmt.Fprintln(c.Output, "Groups:")
	for _, g := range groups {
		fmt.Fprintln(c.Output, "\t", g.GetName(), ": ", g.GetDescription())
	}
	fmt.Fprintln(c.Output, "Services:")
	for _, s := range services {
		fmt.Fprintln(c.Output, "\t", s.GetName(), ": ", s.GetDescription())
	}

	return nil
}

func (c *Client) Log(names []string) error {
	if len(names) == 0 {
		return errors.New("At least one service or group must be specified")
	}
	sgs, err := config.GetServicesOrGroups(names)
	if err != nil {
		return errors.WithStack(err)
	}

	var logChannel = make(chan runner.LogLine)
	var lines []runner.LogLine
	for _, sg := range sgs {
		switch v := sg.(type) {
		case *services.ServiceConfig:
			newLines, err := followServiceLog(v, logChannel)
			if err != nil {
				return err
			}
			lines = append(lines, newLines...)
		case *services.ServiceGroupConfig:
			newLines, err := followGroupLog(v, logChannel)
			if err != nil {
				return err
			}
			lines = append(lines, newLines...)
		}
	}

	// Sort initial lines
	sort.Sort(byTime(lines))
	for _, line := range lines {
		printMessage(line, services.CountServices(sgs) > 1)
	}

	for logMessage := range logChannel {
		printMessage(logMessage, services.CountServices(sgs) > 1)
	}

	return nil
}

func generatorsMatchingTargets(targets []string) ([]generators.Generator, error) {
	allGenerators := []generators.Generator{
		&generators.EdwardGenerator{},
		&generators.DockerGenerator{},
		&generators.GoGenerator{},
		&generators.IcbmGenerator{},
	}
	if len(targets) == 0 {
		return allGenerators, nil
	}

	targetSet := make(map[string]struct{})
	for _, target := range targets {
		targetSet[target] = struct{}{}
	}

	var filteredGenerators = make([]generators.Generator, 0, len(allGenerators))
	for _, gen := range allGenerators {
		if _, exists := targetSet[gen.Name()]; exists {
			filteredGenerators = append(filteredGenerators, gen)
			delete(targetSet, gen.Name())
		}
	}

	if len(targetSet) > 0 {
		var missingTargets = make([]string, 0, len(targetSet))
		for target := range targetSet {
			missingTargets = append(missingTargets, target)
		}
		return nil, fmt.Errorf("targets not found: %v", strings.Join(missingTargets, ", "))

	}

	return filteredGenerators, nil
}

func (c *Client) Generate(names []string, force bool, targets []string) error {
	var cfg config.Config
	configPath := c.Config
	if configPath == "" {
		wd, err := os.Getwd()
		if err == nil {
			configPath = filepath.Join(wd, "edward.json")
		}
	}

	if f, err := os.Stat(configPath); err == nil && f.Size() != 0 {
		cfg, err = config.LoadConfig(configPath, common.EdwardVersion, c.Logger)
		if err != nil {
			return errors.WithMessage(err, configPath)
		}
	} else {
		cfg = config.EmptyConfig(filepath.Dir(configPath), c.Logger)
	}

	wd, err := os.Getwd()
	if err != nil {
		return errors.WithStack(err)
	}

	targetedGenerators, err := generatorsMatchingTargets(targets)
	if err != nil {
		return errors.WithStack(err)
	}

	generators := &generators.GeneratorCollection{
		Generators: targetedGenerators,
		Path:       wd,
		Targets:    names,
	}
	err = generators.Generate()
	if err != nil {
		return errors.WithStack(err)
	}
	foundServices := generators.Services()
	foundGroups := generators.Groups()
	foundImports := generators.Imports()

	if len(foundServices) == 0 && len(foundGroups) == 0 {
		fmt.Println("No services found")
		return nil
	}

	// Prompt user to confirm the list of services that will be generated
	if !force {
		fmt.Println("The following will be generated:")
		if len(foundServices) > 0 {
			fmt.Println("Services:")
		}
		for _, service := range foundServices {
			fmt.Println("\t", service.Name)
		}
		if len(foundGroups) > 0 {
			fmt.Println("Groups:")
		}
		for _, group := range foundGroups {
			fmt.Println("\t", group.Name)
		}
		if len(foundImports) > 0 {
			fmt.Println("Imports:")
		}
		for _, i := range foundImports {
			fmt.Println("\t", i)
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
	err = cfg.AppendGroups(foundGroups)
	if err != nil {
		return errors.WithStack(err)
	}
	cfg.Imports = append(cfg.Imports, foundImports...)

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

func (c *Client) startAndTrack(sgs []services.ServiceOrGroup, skipBuild bool, tail bool, noWatch bool, exclude []string, edwardExecutable string) error {
	cfg := services.OperationConfig{
		EdwardExecutable: edwardExecutable,
		Exclusions:       exclude,
		SkipBuild:        skipBuild,
		NoWatch:          noWatch,
	}

	task := tracker.NewTask(c.Follower.Handle)
	defer c.Follower.Done()

	p := worker.NewPool(1)
	p.Start()
	defer func() {
		p.Stop()
		_ = <-p.Complete()
	}()
	var err error
	for _, s := range sgs {
		if skipBuild {
			err = s.Launch(cfg, services.ContextOverride{}, task, p)
		} else {
			err = s.Start(cfg, services.ContextOverride{}, task, p)
		}
		if err != nil {
			return errors.New("Error launching " + s.GetName() + ": " + err.Error())
		}
	}
	return nil
}

func (c *Client) tailFromFlag(names []string) error {
	fmt.Println("=== Logs ===")
	return errors.WithStack(c.Log(names))
}

func askForConfirmation(question string) bool {

	// Skip confirmations for children. For example, for enabling sudo.
	isChild := os.Getenv("ISCHILD")
	if isChild != "" {
		return true
	}

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

type serviceConfigByPID []*services.ServiceConfig

func (s serviceConfigByPID) Len() int {
	return len(s)
}
func (s serviceConfigByPID) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s serviceConfigByPID) Less(i, j int) bool {
	cmd1, _ := s[i].GetCommand(services.ContextOverride{})
	cmd2, _ := s[j].GetCommand(services.ContextOverride{})
	return cmd1.Pid < cmd2.Pid
}
