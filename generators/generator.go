package generators

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/pkg/errors"
	"github.com/sabhiram/go-git-ignore"
	"github.com/yext/edward/services"
)

type Generator interface {
	Name() string
	StartWalk(basePath string)
	StopWalk()
	VisitDir(path string, f os.FileInfo, err error) error
	Err() error
	SetErr(err error)
}

type ServiceGenerator interface {
	Services() []*services.ServiceConfig
}

type GroupGenerator interface {
	Groups() []*services.ServiceGroupConfig
}

type ImportGenerator interface {
	Imports() []string
}

type generatorBase struct {
	err      error
	basePath string
}

func (e *generatorBase) Err() error {
	return e.err
}

func (e *generatorBase) SetErr(err error) {
	e.err = err
}

func (b *generatorBase) StartWalk(basePath string) {
	b.err = nil
	b.basePath = basePath
}

var Generators []Generator

func RegisterGenerator(g Generator) {
	Generators = append(Generators, g)
}

func loadIgnores(path string, currentIgnores *ignore.GitIgnore) (*ignore.GitIgnore, error) {
	ignoreFile := filepath.Join(path, ".edwardignore")
	if _, err := os.Stat(ignoreFile); err != nil {
		if os.IsNotExist(err) {
			return currentIgnores, nil
		}
		return currentIgnores, errors.WithStack(err)
	}

	ignores, err := ignore.CompileIgnoreFile(ignoreFile)
	return ignores, errors.WithStack(err)
}

func shouldIgnore(basePath, path string, ignores *ignore.GitIgnore) bool {
	if ignores == nil {
		return false
	}

	rel, err := filepath.Rel(basePath, path)
	if err != nil {
		fmt.Println("Error checking ignore:", err)
		return false
	}

	return ignores.MatchesPath(rel)
}

type GeneratorCollection struct {
	Generators []Generator
	Path       string
	Targets    []string
}

func (g *GeneratorCollection) Generate() error {
	if info, err := os.Stat(g.Path); err != nil || !info.IsDir() {
		if err != nil {
			return errors.WithStack(err)
		}
		return errors.New(g.Path + " is not a directory")
	}

	// TODO: Create a stack of ignore files to handle ignores in subdirs
	ignores, err := loadIgnores(g.Path, nil)
	if err != nil {
		return errors.WithStack(err)
	}

	for _, generator := range g.Generators {
		walkGenerator(generator, g.Path, ignores)
		if generator.Err() != nil {
			fmt.Println("Error in generator", generator.Name(), ":", err)
		}
	}

	return nil
}

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
		sort.Sort(ByName(outServices))
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
	sort.Sort(ByName(filteredServices))
	return filteredServices
}

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
		sort.Sort(ByGroupName(outGroups))
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
	sort.Sort(ByGroupName(filteredGroups))
	return filteredGroups
}

func (g *GeneratorCollection) Imports() []string {
	var outImports []string
	for _, generator := range g.Generators {
		if importGenerator, ok := generator.(ImportGenerator); ok && generator.Err() == nil {
			outImports = append(outImports, importGenerator.Imports()...)
		}
	}
	return outImports
}

func walkGenerator(generator Generator, path string, ignores *ignore.GitIgnore) {
	generator.StartWalk(path)
	err := filepath.Walk(path, func(curPath string, f os.FileInfo, err error) error {
		if _, err := os.Stat(curPath); err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return errors.WithStack(err)
		}

		if !f.Mode().IsDir() || shouldIgnore(path, curPath, ignores) {
			return nil
		}

		err = generator.VisitDir(curPath, f, err)
		return errors.WithStack(err)
	})
	if err != nil {
		generator.SetErr(err)
	}
	generator.StopWalk()
}

type ByGroupName []*services.ServiceGroupConfig

func (s ByGroupName) Len() int {
	return len(s)
}
func (s ByGroupName) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s ByGroupName) Less(i, j int) bool {
	return s[i].Name < s[j].Name
}

type ByName []*services.ServiceConfig

func (s ByName) Len() int {
	return len(s)
}
func (s ByName) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s ByName) Less(i, j int) bool {
	return s[i].Name < s[j].Name
}
