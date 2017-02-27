package generators

import (
	"os"
	"sort"

	"github.com/pkg/errors"
	"github.com/yext/edward/services"
)

// GeneratorCollection provides the ability to execute multiple generators under
// the same path.
type GeneratorCollection struct {
	Generators []Generator
	Path       string
	Targets    []string
}

// Generate walks over the directories under the GeneratorCollection's Path and runs
// each generator on each directory in turn. If a generator finds a match, no other generators may
// run in that directory and subdirectories.
func (g *GeneratorCollection) Generate() error {
	if info, err := os.Stat(g.Path); err != nil || !info.IsDir() {
		if err != nil {
			return errors.WithStack(err)
		}
		return errors.New(g.Path + " is not a directory")
	}

	dir, err := newDirectory(g.Path, nil)
	if err != nil {
		return errors.WithStack(err)
	}

	for _, generator := range g.Generators {
		generator.StartWalk(g.Path)
	}
	defer func() {
		for _, generator := range g.Generators {
			generator.StopWalk()
		}
	}()

	return errors.WithStack(dir.Generate(g.Generators))
}

// Services returns any services found during the Generate
func (g *GeneratorCollection) Services() []*services.ServiceConfig {
	var outServices []*services.ServiceConfig
	var serviceToGenerator = make(map[string]string)

	for _, generator := range g.Generators {
		if serviceGenerator, ok := generator.(ServiceGenerator); ok && generator.Err() == nil {
			found := serviceGenerator.Services()
			for _, service := range found {
				serviceToGenerator[service.Name] = generator.Name()
			}
			outServices = append(outServices, found...)
		}
	}

	if len(g.Targets) == 0 {
		sort.Sort(byName(outServices))
		return outServices
	}

	filterMap := make(map[string]struct{})
	for _, name := range g.Targets {
		filterMap[name] = struct{}{}
	}

	var filteredServices []*services.ServiceConfig
	for _, service := range outServices {
		if _, ok := filterMap[service.Name]; ok {
			filteredServices = append(filteredServices, service)
		}
	}
	sort.Sort(byName(filteredServices))
	return filteredServices
}

// Groups returns any groups found during the Generate
func (g *GeneratorCollection) Groups() []*services.ServiceGroupConfig {
	var outGroups []*services.ServiceGroupConfig
	var groupToGenerator = make(map[string]string)

	for _, generator := range g.Generators {
		if groupGenerator, ok := generator.(GroupGenerator); ok && generator.Err() == nil {
			found := groupGenerator.Groups()
			for _, group := range found {
				groupToGenerator[group.Name] = generator.Name()
			}
			outGroups = append(outGroups, found...)
		}
	}

	if len(g.Targets) == 0 {
		sort.Sort(byGroupName(outGroups))
		return outGroups
	}

	filterMap := make(map[string]struct{})
	for _, name := range g.Targets {
		filterMap[name] = struct{}{}
	}

	var filteredGroups []*services.ServiceGroupConfig
	for _, group := range outGroups {
		if _, ok := filterMap[group.Name]; ok {
			filteredGroups = append(filteredGroups, group)
		}
	}
	sort.Sort(byGroupName(filteredGroups))
	return filteredGroups
}

// Imports returns any imports found during the Generate
func (g *GeneratorCollection) Imports() []string {
	var outImports []string
	for _, generator := range g.Generators {
		if importGenerator, ok := generator.(ImportGenerator); ok && generator.Err() == nil {
			outImports = append(outImports, importGenerator.Imports()...)
		}
	}
	return outImports
}

type byGroupName []*services.ServiceGroupConfig

func (s byGroupName) Len() int {
	return len(s)
}
func (s byGroupName) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s byGroupName) Less(i, j int) bool {
	return s[i].Name < s[j].Name
}

type byName []*services.ServiceConfig

func (s byName) Len() int {
	return len(s)
}
func (s byName) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s byName) Less(i, j int) bool {
	return s[i].Name < s[j].Name
}
