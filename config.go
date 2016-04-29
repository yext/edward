package main

import (
	"encoding/json"
	"errors"
	"io"
)

type Config struct {
	Services []ServiceConfig `json:"services"`
	Groups   []GroupDef      `json:"groups"`
}

type GroupDef struct {
	Name     string   `json:"name"`
	Children []string `json:"children"`
}

func NewConfig(services []ServiceConfig, groups []ServiceGroupConfig) Config {
	cfg := Config{
		Services: services,
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
			grp.Children = append(grp.Children, cs.Name)
		}
		cfg.Groups = append(cfg.Groups, grp)
	}

	return cfg
}

func (c Config) BuildGroupConfig() ([]ServiceGroupConfig, error) {
	// TODO: Implement
	return []ServiceGroupConfig{}, errors.New("Not implemented")
}

// Reader from os.Open
func LoadConfig(reader io.Reader) (Config, error) {
	var config Config
	dec := json.NewDecoder(reader)
	err := dec.Decode(&config)
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
