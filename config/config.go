package config

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/yext/edward/services"
	"github.com/yext/errgo"
)

type Config struct {
	workingDir       string                   `json:"-"`
	Imports          []string                 `json:"imports"`
	ImportedGroups   []GroupDef               `json:"-"`
	ImportedServices []services.ServiceConfig `json:"-"`
	Env              []string                 `json:"env"`
	Groups           []GroupDef               `json:"groups"`
	Services         []services.ServiceConfig `json:"services"`

	ServiceMap map[string]*services.ServiceConfig      `json:"-"`
	GroupMap   map[string]*services.ServiceGroupConfig `json:"-"`
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

func NewConfig(newServices []services.ServiceConfig, newGroups []services.ServiceGroupConfig) Config {

	// Find Env settings common to all services
	var allEnvSlices [][]string
	for _, s := range newServices {
		allEnvSlices = append(allEnvSlices, s.Env)
	}
	env := stringSliceIntersect(allEnvSlices)

	// Remove common settings from services
	var svcs []services.ServiceConfig
	for _, s := range newServices {
		s.Env = stringSliceRemoveCommon(env, s.Env)
		svcs = append(svcs, s)
	}

	cfg := Config{
		Env:      env,
		Services: svcs,
		Groups:   []GroupDef{},
	}

	cfg.AddGroups(newGroups)

	return cfg
}

func EmptyConfig(workingDir string) Config {

	cfg := Config{
		workingDir: workingDir,
	}

	cfg.ServiceMap = make(map[string]*services.ServiceConfig)
	cfg.GroupMap = make(map[string]*services.ServiceGroupConfig)

	return cfg
}

// AppendServices adds services to an existing config without replacing existing services
func (cfg *Config) AppendServices(newServices []*services.ServiceConfig) error {
	if cfg.ServiceMap == nil {
		cfg.ServiceMap = make(map[string]*services.ServiceConfig)
	}
	for _, s := range newServices {
		if _, found := cfg.ServiceMap[s.Name]; !found {
			cfg.ServiceMap[s.Name] = s
			cfg.Services = append(cfg.Services, *s)
		}
	}
	return nil
}

func (cfg *Config) AddGroups(groups []services.ServiceGroupConfig) error {
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
	return nil
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
	var svcs map[string]*services.ServiceConfig = make(map[string]*services.ServiceConfig)
	for _, s := range append(c.Services, c.ImportedServices...) {
		sc := s
		sc.Env = append(sc.Env, c.Env...)
		if _, exists := svcs[s.Name]; exists {
			return errgo.New("Service name already exists: " + s.Name)
		}
		svcs[s.Name] = &sc
	}
	c.ServiceMap = svcs

	var groups map[string]*services.ServiceGroupConfig = make(map[string]*services.ServiceGroupConfig)
	// First pass: Services
	var orphanNames map[string]struct{} = make(map[string]struct{})
	for _, g := range append(c.Groups, c.ImportedGroups...) {

		var childServices []*services.ServiceConfig

		for _, name := range g.Children {
			if s, ok := svcs[name]; ok {
				childServices = append(childServices, s)
			} else {
				orphanNames[name] = struct{}{}
			}
		}

		groups[g.Name] = &services.ServiceGroupConfig{
			Name:     g.Name,
			Services: childServices,
			Groups:   []*services.ServiceGroupConfig{},
		}
	}

	// Second pass: Groups
	for _, g := range append(c.Groups, c.ImportedGroups...) {
		childGroups := []*services.ServiceGroupConfig{}

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
