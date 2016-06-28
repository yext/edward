package main

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/yext/errgo"
)

type Config struct {
	workingDir       string          `json:"-"`
	Imports          []string        `json:"imports"`
	ImportedGroups   []GroupDef      `json:"-"`
	ImportedServices []ServiceConfig `json:"-"`
	Env              []string        `json:"env"`
	Groups           []GroupDef      `json:"groups"`
	Services         []ServiceConfig `json:"services"`

	ServiceMap map[string]*ServiceConfig      `json:"-"`
	GroupMap   map[string]*ServiceGroupConfig `json:"-"`
}

type GroupDef struct {
	Name     string   `json:"name"`
	Children []string `json:"children"`
}

func LoadConfig(reader io.Reader) (Config, error) {
	outCfg, err := LoadConfigWithDir(reader, "")
	return outCfg, errgo.Mask(err)
}

func LoadConfigWithDir(reader io.Reader, workingDir string) (Config, error) {
	config, err := LoadConfigContents(reader, workingDir)
	if err != nil {
		return Config{}, errgo.Mask(err)
	}
	err = config.initMaps()
	return config, errgo.Mask(err)
}

// Reader from os.Open
func LoadConfigContents(reader io.Reader, workingDir string) (Config, error) {
	var config Config
	dec := json.NewDecoder(reader)
	err := dec.Decode(&config)

	if err != nil {
		return Config{}, errgo.Mask(err)
	}

	config.workingDir = workingDir

	err = config.loadImports()
	if err != nil {
		return Config{}, errgo.Mask(err)
	}

	return config, nil
}

func (c Config) Save(writer io.Writer) error {
	content, err := json.MarshalIndent(c, "", "    ")
	if err != nil {
		return err
	}
	_, err = writer.Write(content)
	return err
}

func NewConfig(services []ServiceConfig, groups []ServiceGroupConfig) Config {

	// Find Env settings common to all services
	var allEnvSlices [][]string
	for _, s := range services {
		allEnvSlices = append(allEnvSlices, s.Env)
	}
	env := stringSliceIntersect(allEnvSlices)

	// Remove common settings from services
	var svcs []ServiceConfig
	for _, s := range services {
		s.Env = stringSliceRemoveCommon(env, s.Env)
		svcs = append(svcs, s)
	}

	cfg := Config{
		Env:      env,
		Services: svcs,
		Groups:   []GroupDef{},
	}

	for _, group := range groups {
		grp := GroupDef{
			Name:     group.Name,
			Children: []string{},
		}
		for _, cg := range group.Groups {
			if cg != nil {
				grp.Children = append(grp.Children, cg.Name)
			}
		}
		for _, cs := range group.Services {
			if cs != nil {
				grp.Children = append(grp.Children, cs.Name)
			}
		}
		cfg.Groups = append(cfg.Groups, grp)
	}

	return cfg
}

func (c *Config) loadImports() error {
	for _, i := range c.Imports {
		var cPath string
		if filepath.IsAbs(i) {
			cPath = i
		} else {
			cPath = filepath.Join(c.workingDir, i)
		}

		r, err := os.Open(cPath)
		if err != nil {
			return errgo.Mask(err)
		}
		cfg, err := LoadConfigContents(r, filepath.Dir(cPath))
		if err != nil {
			return errgo.Mask(err)
		}

		err = c.importConfig(cfg)
		if err != nil {
			return errgo.Mask(err)
		}
	}
	return nil
}

func (c *Config) importConfig(second Config) error {
	for _, service := range second.Services {
		c.ImportedServices = append(c.ImportedServices, service)
	}
	for _, group := range second.Groups {
		c.ImportedGroups = append(c.ImportedGroups, group)
	}
	return nil
}

func (c *Config) initMaps() error {
	var services map[string]*ServiceConfig = make(map[string]*ServiceConfig)
	for _, s := range append(c.Services, c.ImportedServices...) {
		sc := s
		sc.Env = append(sc.Env, c.Env...)
		if _, exists := services[s.Name]; exists {
			return errgo.New("Service name already exists: " + s.Name)
		}
		services[s.Name] = &sc
	}
	c.ServiceMap = services

	var groups map[string]*ServiceGroupConfig = make(map[string]*ServiceGroupConfig)
	// First pass: Services
	var orphanNames map[string]struct{} = make(map[string]struct{})
	for _, g := range append(c.Groups, c.ImportedGroups...) {

		var childServices []*ServiceConfig

		for _, name := range g.Children {
			if s, ok := services[name]; ok {
				childServices = append(childServices, s)
			} else {
				orphanNames[name] = struct{}{}
			}
		}

		groups[g.Name] = &ServiceGroupConfig{
			Name:     g.Name,
			Services: childServices,
			Groups:   []*ServiceGroupConfig{},
		}
	}

	// Second pass: Groups
	for _, g := range append(c.Groups, c.ImportedGroups...) {
		childGroups := []*ServiceGroupConfig{}

		for _, name := range g.Children {
			if gr, ok := groups[name]; ok {
				delete(orphanNames, name)
				childGroups = append(childGroups, gr)
			}
		}
		groups[g.Name].Groups = childGroups
	}

	if len(orphanNames) > 0 {
		var keys []string
		for k := range orphanNames {
			keys = append(keys, k)
		}
		return errgo.New("A service or group could not be found for the following names: " + strings.Join(keys, ", "))
	}

	c.GroupMap = groups
	return nil
}

func stringSliceIntersect(slices [][]string) []string {
	var counts map[string]int = make(map[string]int)
	for _, s := range slices {
		for _, v := range s {
			counts[v] += 1
		}
	}

	var outSlice []string
	for v, count := range counts {
		if count == len(slices) {
			outSlice = append(outSlice, v)
		}
	}
	return outSlice
}

func stringSliceRemoveCommon(common []string, original []string) []string {
	var commonMap map[string]interface{} = make(map[string]interface{})
	for _, s := range common {
		commonMap[s] = struct{}{}
	}
	var outSlice []string
	for _, s := range original {
		if _, ok := commonMap[s]; !ok {
			outSlice = append(outSlice, s)
		}
	}
	return outSlice
}
