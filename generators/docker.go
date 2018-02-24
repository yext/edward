package generators

import (
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/yext/edward/services"
)

// DockerGenerator generates services from Docker files.
// Services are generated from a Dockerfile.
//
// The container is build with a tag based on the directory name suffixed with ':edward'.
// For a Dockerfile under 'service', the tag would be 'service:edward'.
//
// Ports identified with EXPOSE in the Dockerfile will be forwarded from the container,
// with the local port matching the port in the container.
type DockerGenerator struct {
	generatorBase
	foundServices []*services.ServiceConfig
}

// Name returns 'docker' to identify this generator
func (v *DockerGenerator) Name() string {
	return "docker"
}

// VisitDir searches a directory for a Dockerfile and stores a service configuration if
// one is found. Returns true in the first return value if a service was found.
func (v *DockerGenerator) VisitDir(path string) (bool, error) {
	files, _ := ioutil.ReadDir(path)
	for _, f := range files {
		if f.Name() != "Dockerfile" {
			continue
		}

		dockerPath, err := filepath.Rel(v.basePath, path)
		if err != nil {
			return false, errors.WithStack(err)
		}

		fPath := filepath.Join(path, f.Name())
		expectedPorts, portCommands, err := getPorts(fPath)
		if err != nil {
			return false, errors.WithStack(err)
		}

		name := filepath.Base(path)
		tag := name + ":edward"
		service := &services.ServiceConfig{
			Name: name,
			Path: &dockerPath,
			Env:  []string{},
			TypeConfig: &services.ConfigCommandLine{
				Commands: services.ServiceConfigCommands{
					Build:  "docker build -t " + tag + " .",
					Launch: "docker run " + strings.Join(portCommands, " ") + " " + tag,
				},
			},
			LaunchChecks: &services.LaunchChecks{
				Ports: expectedPorts,
			},
		}
		v.foundServices = append(v.foundServices, service)
		return true, nil
	}

	return false, nil
}

func getPorts(dockerFilePath string) ([]int, []string, error) {
	input, err := ioutil.ReadFile(dockerFilePath)
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}
	var ports []int
	var portCommands []string
	exposeExpr := regexp.MustCompile(`(?m)^(?:EXPOSE )([0-9]+)$`)
	for _, match := range exposeExpr.FindAllStringSubmatch(string(input), -1) {
		portCommands = append(portCommands, "-p "+match[1]+":"+match[1])
		port, err := strconv.Atoi(match[1])
		if err != nil {
			return nil, nil, errors.WithStack(err)
		}
		ports = append(ports, port)
	}
	return ports, portCommands, nil
}

// Services returns a slice of services identified in the directory walk
func (v *DockerGenerator) Services() []*services.ServiceConfig {
	return v.foundServices
}
