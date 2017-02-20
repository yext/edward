package generators

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/yext/edward/services"
)

type ProcfileGenerator struct {
	generatorBase
	foundGroups   []*services.ServiceGroupConfig
	foundServices []*services.ServiceConfig
}

func (v *ProcfileGenerator) Name() string {
	return "procfile"
}

func (v *ProcfileGenerator) StopWalk() {
}

func (v *ProcfileGenerator) VisitDir(path string) (bool, error) {
	procfilePath := filepath.Join(path, "Procfile")

	if _, err := os.Stat(procfilePath); os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, errors.WithStack(err)
	}

	relPath, err := filepath.Rel(v.basePath, path)
	if err != nil {
		return false, errors.WithStack(err)
	}

	specFile, err := os.Open(procfilePath)
	if err != nil {
		return false, errors.WithStack(err)
	}

	group := &services.ServiceGroupConfig{
		Name: filepath.Base(path),
	}

	scanner := bufio.NewScanner(specFile)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, ":") {
			def := strings.SplitN(line, ":", 2)
			service := &services.ServiceConfig{
				Name: group.Name + "-" + def[0],
				Path: &relPath,
				Commands: services.ServiceConfigCommands{
					Launch: strings.TrimSpace(def[1]),
				},
			}
			group.Services = append(group.Services, service)
		}
	}
	if err := scanner.Err(); err != nil {
		return false, errors.WithStack(err)
	}
	v.foundServices = append(v.foundServices, group.Services...)
	v.foundGroups = append(v.foundGroups, group)
	return true, nil
}

func (v *ProcfileGenerator) Groups() []*services.ServiceGroupConfig {
	return v.foundGroups
}

func (v *ProcfileGenerator) Services() []*services.ServiceConfig {
	return v.foundServices
}
