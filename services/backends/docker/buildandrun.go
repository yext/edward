package docker

import (
	"fmt"
	"io"
	"strings"
	"sync"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/pkg/errors"
	"github.com/theothertomelliott/gopsutil-nocgo/process"
	"github.com/yext/edward/services"
)

type buildandrun struct {
	Service *services.ServiceConfig
	Backend *Backend

	containerID string
	client      *docker.Client

	done chan struct{}

	mtx sync.Mutex
}

var _ services.Builder = &buildandrun{}
var _ services.Runner = &buildandrun{}

func (b *buildandrun) Build(string, func(string) string) ([]byte, error) {
	return nil, errors.New("no build step for docker")
}

func (b *buildandrun) Start(standardLog io.Writer, errorLog io.Writer) error {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	b.done = make(chan struct{})

	var err error
	b.client, err = docker.NewClientFromEnv()
	if err != nil {
		return errors.WithMessage(err, "initializing client")
	}

	imgID, err := b.findImage(standardLog)
	if err != nil {
		return errors.WithMessage(err, "getting image")
	}

	var running bool
	b.containerID, running, err = b.findContainer()
	if err != nil {
		return errors.WithMessage(err, "finding container id")
	}

	exposedPorts, portBindings := b.formatPortMappings()

	if b.containerID == "" {
		container, err := b.client.CreateContainer(docker.CreateContainerOptions{
			Name: b.containerName(),
			Config: &docker.Config{
				Image:        imgID,
				ExposedPorts: exposedPorts,
			},
			HostConfig: &docker.HostConfig{
				PortBindings: portBindings,
			},
		})
		if err != nil {
			return errors.WithMessage(err, "creating container")
		}
		b.containerID = container.ID
	}

	if running {
		return errors.New("already running")
	}

	err = b.client.StartContainer(b.containerID, &docker.HostConfig{})
	if err != nil {
		return errors.WithMessage(err, "starting container")
	}

	_, err = b.client.AttachToContainerNonBlocking(docker.AttachToContainerOptions{
		Container:    b.containerID,
		Stream:       true,
		Stdout:       true,
		Stderr:       true,
		OutputStream: standardLog,
		ErrorStream:  errorLog,
	})
	if err != nil {
		return errors.WithMessage(err, "attaching to container")
	}
	return nil
}

func (b *buildandrun) Stop(workingDir string, getenv func(string) string) ([]byte, error) {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	err := b.client.StopContainer(b.containerID, 0)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	close(b.done)
	return nil, nil
}

func (b *buildandrun) Status() (services.BackendStatus, error) {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	var ports []string
	for _, mappedPort := range b.Backend.Ports {
		ports = append(ports, mappedPort)
	}

	return services.BackendStatus{
		MemoryInfo: &process.MemoryInfoStat{},
		Ports:      ports,
	}, nil
}

func (b *buildandrun) Wait() {
	<-b.done
}

func (b *buildandrun) containerName() string {
	return fmt.Sprintf("edward-%s", b.Service.Name)
}

func (b *buildandrun) findImage(standardLog io.Writer) (string, error) {
	tag := b.Backend.Tag
	if tag == "" {
		tag = "latest"
	}

	err := b.client.PullImage(docker.PullImageOptions{
		Repository:   b.Backend.Repository,
		Tag:          tag,
		OutputStream: standardLog,
	}, docker.AuthConfiguration{})
	if err != nil {
		errors.WithMessage(err, "pulling image")
	}

	imgs, err := b.client.ListImages(docker.ListImagesOptions{All: false})
	if err != nil {
		return "", errors.WithStack(err)
	}
	var imgID string
	for _, img := range imgs {
		if len(img.RepoTags) > 0 && strings.Contains(img.RepoTags[0], fmt.Sprintf("%s:%s", b.Backend.Repository, tag)) {
			imgID = img.ID
		}
	}
	return imgID, nil
}

func (b *buildandrun) findContainer() (string, bool, error) {
	var containerID string

	containers, err := b.client.ListContainers(docker.ListContainersOptions{
		All: true,
	})
	if err != nil {
		return "", false, errors.WithStack(err)
	}

	var running bool
	for _, container := range containers {
		for _, name := range container.Names {
			if name == fmt.Sprintf("/%s", b.containerName()) {
				containerID = container.ID
				running = container.State == "running"
			}
		}
	}
	return containerID, running, nil
}

func (b *buildandrun) formatPortMappings() (exposedPorts map[docker.Port]struct{}, portBindings map[docker.Port][]docker.PortBinding) {
	exposedPorts = make(map[docker.Port]struct{})
	portBindings = make(map[docker.Port][]docker.PortBinding)

	getDockerPort := func(port string) docker.Port {
		if strings.Contains(port, "/") {
			return docker.Port(port)
		}
		return docker.Port(fmt.Sprintf("%s/tcp", port))
	}

	for port, mapping := range b.Backend.Ports {
		dPort := getDockerPort(port)
		mapPort := getDockerPort(mapping)
		exposedPorts[dPort] = struct{}{}
		portBindings[dPort] = []docker.PortBinding{
			docker.PortBinding{
				HostPort: string(mapPort),
			},
		}
	}

	return exposedPorts, portBindings
}
