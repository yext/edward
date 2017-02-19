package generators

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/yext/edward/services"
)

func init() {
	RegisterGenerator(&DockerGenerator{})
}

type DockerGenerator struct {
	generatorBase
	foundServices []*services.ServiceConfig
}

func (v *DockerGenerator) Name() string {
	return "docker"
}

func (v *DockerGenerator) StopWalk() {
}

func (v *DockerGenerator) VisitDir(path string, f os.FileInfo, err error) error {
	if err != nil {
		return errors.WithStack(err)
	}

	if _, err := os.Stat(path); err != nil {
		return errors.WithStack(err)
	}

	files, _ := ioutil.ReadDir(path)
	for _, f := range files {
		if f.Name() != "Dockerfile" {
			continue
		}

		dockerPath, err := filepath.Rel(v.basePath, path)
		if err != nil {
			return errors.WithStack(err)
		}

		fPath := filepath.Join(path, f.Name())
		expectedPorts, portCommands, err := getPorts(fPath)
		if err != nil {
			return errors.WithStack(err)
		}

		name := filepath.Base(path)
		tag := name + ":edward"
		service := &services.ServiceConfig{
			Name: name,
			Path: &dockerPath,
			Env:  []string{},
			Commands: services.ServiceConfigCommands{
				Build:  "docker build -t " + tag + " .",
				Launch: "docker run " + strings.Join(portCommands, " ") + " " + tag,
			},
			LaunchChecks: &services.LaunchChecks{
				Ports: expectedPorts,
			},
		}
		v.foundServices = append(v.foundServices, service)
		break
	}

	return nil
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

func (v *DockerGenerator) Services() []*services.ServiceConfig {
	return v.foundServices
}
