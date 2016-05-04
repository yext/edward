package main

import (
	"encoding/json"
	"io"
)

type Config struct {
	Env      []string        `json:"env"`
	Groups   []GroupDef      `json:"groups"`
	Services []ServiceConfig `json:"services"`

	ServiceMap map[string]*ServiceConfig      `json:"-"`
	GroupMap   map[string]*ServiceGroupConfig `json:"-"`
}

type GroupDef struct {
	Name     string   `json:"name"`
	Children []string `json:"children"`
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

func (c *Config) initMaps() {
	var services map[string]*ServiceConfig = make(map[string]*ServiceConfig)
	for _, s := range c.Services {
		sc := s
		sc.Env = append(sc.Env, c.Env...)
		services[s.Name] = &sc
	}
	c.ServiceMap = services

	var groups map[string]*ServiceGroupConfig = make(map[string]*ServiceGroupConfig)
	// First pass: Services
	for _, g := range c.Groups {

		childServices := []*ServiceConfig{}

		for _, name := range g.Children {
			if s, ok := services[name]; ok {
				childServices = append(childServices, s)
			}
		}

		groups[g.Name] = &ServiceGroupConfig{
			Name:     g.Name,
			Services: childServices,
			Groups:   []*ServiceGroupConfig{},
		}
	}

	// Second pass: Groups
	for _, g := range c.Groups {
		childGroups := []*ServiceGroupConfig{}

		for _, name := range g.Children {
			if gr, ok := groups[name]; ok {
				childGroups = append(childGroups, gr)
			}
		}
		groups[g.Name].Groups = childGroups
	}

	c.GroupMap = groups
}

// Reader from os.Open
func LoadConfig(reader io.Reader) (Config, error) {
	var config Config
	dec := json.NewDecoder(reader)
	err := dec.Decode(&config)

	config.initMaps()

	return config, err
}

func (c Config) Save(writer io.Writer) error {
	content, err := json.MarshalIndent(c, "", "    ")
	if err != nil {
		return err
	}
	_, err = writer.Write(content)
	return err
}
