package services

import (
	"encoding/json"
	"io/ioutil"
	"path"

	"github.com/pkg/errors"
	"github.com/yext/edward/home"
)

func LoadRunningServices() ([]ServiceOrGroup, error) {
	dir := home.EdwardConfig.StateDir
	stateFiles, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	var services []ServiceOrGroup
	for _, file := range stateFiles {
		command := &ServiceCommand{}
		raw, err := ioutil.ReadFile(path.Join(dir, file.Name()))
		if err != nil {
			return nil, errors.WithStack(err)
		}
		json.Unmarshal(raw, command)
		command.Service.ConfigFile = command.ConfigFile

		// Check this service is actually running
		valid, err := command.validateState()
		if err != nil {
			return nil, errors.WithStack(err)
		}

		if valid {
			services = append(services, command.Service)
		}
	}
	return services, nil
}
