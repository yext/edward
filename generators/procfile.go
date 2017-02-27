package generators

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/yext/edward/services"
)

// ProcfileGenerator generates services and groups from Procfiles.
//
// For each Procfile, a group is generated to contain a set of services, one per
// process in the Procfile.
//
// The group is named for the directory containing the Procfile, services are named
// using the form '[group]:[process]'.
type ProcfileGenerator struct {
	generatorBase
	foundGroups   []*services.ServiceGroupConfig
	foundServices []*services.ServiceConfig
}

// Name returns 'procfile' to identify this generator
func (v *ProcfileGenerator) Name() string {
	return "procfile"
}

// VisitDir searches a directory for a Procfile, generating services and groups for any
// found. Returns true in the first return value if a Procfile was found.
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

// Groups returns a slice of groups generated on previous walks
func (v *ProcfileGenerator) Groups() []*services.ServiceGroupConfig {
	return v.foundGroups
}

// Services returns a slice of services generated on previous walks
func (v *ProcfileGenerator) Services() []*services.ServiceConfig {
	return v.foundServices
}
