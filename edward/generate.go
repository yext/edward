package edward

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/yext/edward/common"
	"github.com/yext/edward/config"
	"github.com/yext/edward/generators"
	"github.com/yext/edward/services"
)

func (c *Client) Generate(names []string, force bool, group string, targets []string) error {
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

	if len(foundServices) == 0 &&
		len(foundGroups) == 0 &&
		len(foundImports) == 0 {
		fmt.Fprintln(c.Output, "No services, groups or imports found")
		return nil
	}

	filteredServices, filteredGroups, filteredImports, err := c.filterGenerated(
		&cfg,
		foundServices,
		foundGroups,
		foundImports,
	)
	if err != nil {
		return errors.WithStack(err)
	}
	if len(filteredServices) == 0 &&
		len(filteredGroups) == 0 &&
		len(filteredImports) == 0 {
		fmt.Fprintln(c.Output, "No new services, groups or imports found")
		return nil
	}

	// Check for duplicates
	duplicates := findDuplicates(append(filteredServices, filteredGroups...))
	if len(duplicates) > 0 {
		return errors.New(fmt.Sprint("Multiple services or groups were found with the names: ", strings.Join(duplicates, ", ")))
	}

	// Prompt user to confirm the list of services that will be generated
	if !force {
		confirmed, err := c.confirmList(&cfg, filteredServices, filteredGroups, filteredImports)
		if !confirmed {
			return errors.WithStack(err)
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

	// Put all discovered services into the specified group
	if len(group) > 0 {
		var newGroupConfig *services.ServiceGroupConfig
		if existingGroup, ok := cfg.GroupMap[group]; ok {
			newGroupConfig = existingGroup
			err := cfg.RemoveGroup(group)
			if err != nil {
				return errors.WithStack(err)
			}
		} else {
			newGroupConfig = &services.ServiceGroupConfig{
				Name: group,
			}
		}
		for _, s := range filteredServices {
			newGroupConfig.Services = append(newGroupConfig.Services, cfg.ServiceMap[s])
		}
		for _, g := range filteredGroups {
			newGroupConfig.Groups = append(newGroupConfig.Groups, cfg.GroupMap[g])
		}
		cfg.AppendGroups([]*services.ServiceGroupConfig{newGroupConfig})
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

	fmt.Fprintln(c.Output, "Wrote to:", configPath)

	return nil
}

func findDuplicates(s []string) []string {
	found := make(map[string]struct{})
	duplicates := make(map[string]struct{})
	for _, name := range s {
		if _, ok := found[name]; ok {
			duplicates[name] = struct{}{}
		}
		found[name] = struct{}{}
	}
	var dupList []string
	for name := range duplicates {
		dupList = append(dupList, name)
	}
	return dupList
}

func (c *Client) filterGenerated(cfg *config.Config,
	foundServices []*services.ServiceConfig,
	foundGroups []*services.ServiceGroupConfig,
	foundImports []string) ([]string, []string, []string, error) {
	var filteredServices []string
	for _, service := range foundServices {
		if _, ok := cfg.ServiceMap[service.Name]; !ok {
			filteredServices = append(filteredServices, service.Name)
		}
	}
	var filteredGroups []string
	for _, group := range foundGroups {
		if _, ok := cfg.GroupMap[group.Name]; !ok {
			filteredGroups = append(filteredGroups, group.Name)
		}
	}
	var filteredImports []string
	for _, i := range foundImports {
		var found bool
		for _, existingImport := range cfg.Imports {
			if existingImport == i {
				found = true
			}
		}
		if !found {
			filteredImports = append(filteredImports, i)
		}
	}
	return filteredServices, filteredGroups, filteredImports, nil
}

func (c *Client) confirmList(cfg *config.Config,
	filteredServices []string,
	filteredGroups []string,
	filteredImports []string) (bool, error) {

	fmt.Fprintln(c.Output, "The following will be generated:")
	if len(filteredServices) > 0 {
		fmt.Fprintln(c.Output, "Services:")
	}
	for _, service := range filteredServices {
		fmt.Fprintf(c.Output, "\t%v\n", service)
	}
	if len(filteredGroups) > 0 {
		fmt.Fprintln(c.Output, "Groups:")
	}
	for _, group := range filteredGroups {
		fmt.Fprintf(c.Output, "\t%v\n", group)
	}
	if len(filteredImports) > 0 {
		fmt.Fprintln(c.Output, "Imports:")
	}
	for _, i := range filteredImports {
		fmt.Fprintf(c.Output, "\t%v\n", i)
	}

	if !c.askForConfirmation("Do you wish to continue?") {
		return false, nil
	}

	return true, nil
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
