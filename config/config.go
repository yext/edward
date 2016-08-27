package config

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/juju/errgo"
	"github.com/yext/edward/common"
	"github.com/yext/edward/services"
)

type Config struct {
	workingDir       string
	MinEdwardVersion string                   `json:"edwardVersion,omitempty"`
	Imports          []string                 `json:"imports,omitempty"`
	ImportedGroups   []GroupDef               `json:"-"`
	ImportedServices []services.ServiceConfig `json:"-"`
	Env              []string                 `json:"env,omitempty"`
	Groups           []GroupDef               `json:"groups,omitempty"`
	Services         []services.ServiceConfig `json:"services"`

	ServiceMap map[string]*services.ServiceConfig      `json:"-"`
	GroupMap   map[string]*services.ServiceGroupConfig `json:"-"`

	Logger common.Logger `json:"-"`
}

type GroupDef struct {
	Name     string   `json:"name"`
	Children []string `json:"children"`
}

func LoadConfig(reader io.Reader, logger common.Logger) (Config, error) {
	outCfg, err := LoadConfigWithDir(reader, "", logger)
	return outCfg, errgo.Mask(err)
}

func LoadConfigWithDir(reader io.Reader, workingDir string, logger common.Logger) (Config, error) {
	config, err := loadConfigContents(reader, workingDir, logger)
	if err != nil {
		return Config{}, errgo.Mask(err)
	}
	err = config.initMaps()

	config.printf("Config loaded with: %d groups and %d services\n", len(config.GroupMap), len(config.ServiceMap))

	return config, errgo.Mask(err)
}

// Reader from os.Open
func loadConfigContents(reader io.Reader, workingDir string, logger common.Logger) (Config, error) {
	log := common.MaskLogger(logger)
	log.Printf("Loading config with working dir %v.\n", workingDir)

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

	config.Logger = log

	return config, nil
}

func (c Config) Save(writer io.Writer) error {
	c.printf("Saving config")
	content, err := json.MarshalIndent(c, "", "    ")
	if err != nil {
		return err
	}
	_, err = writer.Write(content)
	return err
}

func NewConfig(newServices []services.ServiceConfig, newGroups []services.ServiceGroupConfig, logger common.Logger) Config {

	log := common.MaskLogger(logger)
	log.Printf("Creating new config with %d services and %d groups.\n", len(newServices), len(newGroups))

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
		Logger:   log,
	}

	cfg.AddGroups(newGroups)

	log.Printf("Config created: %v", cfg)

	return cfg
}

func EmptyConfig(workingDir string, logger common.Logger) Config {

	log := common.MaskLogger(logger)
	log.Printf("Creating empty config\n")

	cfg := Config{
		workingDir: workingDir,
		Logger:     log,
	}

	cfg.ServiceMap = make(map[string]*services.ServiceConfig)
	cfg.GroupMap = make(map[string]*services.ServiceGroupConfig)

	return cfg
}

// NormalizeServicePaths will modify the Paths for each of the provided services
// to be relative to the working directory of this config file
func (cfg *Config) NormalizeServicePaths(searchPath string, newServices []*services.ServiceConfig) ([]*services.ServiceConfig, error) {
	cfg.printf("Normalizing paths for %d services.\n", len(newServices))
	var outServices []*services.ServiceConfig
	for _, s := range newServices {
		curService := *s
		fullPath := filepath.Join(searchPath, *curService.Path)
		relPath, err := filepath.Rel(cfg.workingDir, fullPath)
		if err != nil {
			return outServices, errgo.Mask(err)
		}
		curService.Path = &relPath
		outServices = append(outServices, &curService)
	}
	return outServices, nil
}

// AppendServices adds services to an existing config without replacing existing services
func (cfg *Config) AppendServices(newServices []*services.ServiceConfig) error {
	cfg.printf("Appending %d services.\n", len(newServices))
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
	cfg.printf("Adding %d groups.\n", len(groups))
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
	c.printf("Loading imports\n")
	for _, i := range c.Imports {
		var cPath string
		if filepath.IsAbs(i) {
			cPath = i
		} else {
			cPath = filepath.Join(c.workingDir, i)
		}

		c.printf("Loading: %v\n", cPath)

		r, err := os.Open(cPath)
		if err != nil {
			return errgo.Mask(err)
		}
		cfg, err := loadConfigContents(r, filepath.Dir(cPath), c.Logger)
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

func (c *Config) combinePath(path string) *string {
	if filepath.IsAbs(path) || strings.HasPrefix(path, "$") {
		return &path
	}
	fullPath := filepath.Join(c.workingDir, path)
	return &fullPath
}

func (c *Config) initMaps() error {
	var svcs map[string]*services.ServiceConfig = make(map[string]*services.ServiceConfig)
	var servicesSkipped = make(map[string]struct{})
	for _, s := range append(c.Services, c.ImportedServices...) {
		sc := s
		sc.Logger = c.Logger
		sc.Env = append(sc.Env, c.Env...)
		if _, exists := svcs[sc.Name]; exists {
			return errgo.New("Service name already exists: " + sc.Name)
		}
		if sc.MatchesPlatform() {
			svcs[sc.Name] = &sc
		} else {
			servicesSkipped[sc.Name] = struct{}{}
		}
	}

	var groups map[string]*services.ServiceGroupConfig = make(map[string]*services.ServiceGroupConfig)
	// First pass: Services
	var orphanNames map[string]struct{} = make(map[string]struct{})
	for _, g := range append(c.Groups, c.ImportedGroups...) {
		var childServices []*services.ServiceConfig

		for _, name := range g.Children {
			if s, ok := svcs[name]; ok {
				if s.Path != nil {
					s.Path = c.combinePath(*s.Path)
				}
				childServices = append(childServices, s)
			} else if _, skipped := servicesSkipped[name]; !skipped {
				orphanNames[name] = struct{}{}
			}
		}

		if _, exists := groups[g.Name]; exists {
			return errgo.New("Group name already exists: " + g.Name)
		}

		groups[g.Name] = &services.ServiceGroupConfig{
			Name:     g.Name,
			Services: childServices,
			Groups:   []*services.ServiceGroupConfig{},
			Logger:   c.Logger,
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
			if hasChildCycle(groups[g.Name], childGroups) {
				return errgo.New("group cycle: " + g.Name)
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

	c.ServiceMap = svcs
	c.GroupMap = groups
	return nil
}

func hasChildCycle(parent *services.ServiceGroupConfig, children []*services.ServiceGroupConfig) bool {
	for _, sg := range children {
		if parent == sg {
			return true
		}
		if hasChildCycle(parent, sg.Groups) {
			return true
		}
	}
	return false
}

func (c *Config) printf(format string, v ...interface{}) {
	if c.Logger == nil {
		return
	}
	c.Logger.Printf(format, v...)
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
