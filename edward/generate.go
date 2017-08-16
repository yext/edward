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
)

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
