package docker

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	"github.com/theothertomelliott/gopsutil-nocgo/process"
	"github.com/theothertomelliott/struct2struct"
	"github.com/yext/edward/services"
)

type buildandrun struct {
	Service *services.ServiceConfig
	Backend *Backend

	containerID string
	client      *client.Client

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
	b.client, err = client.NewEnvClient()
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

	if b.containerID == "" {

		backendConfig := b.Backend.ContainerConfig

		var config container.Config
		var hostConfig container.HostConfig

		struct2struct.Marshal(&backendConfig, &config)
		struct2struct.Marshal(&b.Backend.HostConfig, &hostConfig)

		config.Image = imgID

		container, err := b.client.ContainerCreate(
			context.TODO(),
			&config,
			&hostConfig,
			&network.NetworkingConfig{},
			b.containerName(),
		)

		if err != nil {
			return errors.WithMessage(err, "creating container")
		}
		b.containerID = container.ID
	}

	if running {
		return errors.New("already running")
	}

	err = b.client.ContainerStart(context.TODO(), b.containerID, types.ContainerStartOptions{})
	if err != nil {
		return errors.WithMessage(err, "starting container")
	}

	response, err := b.client.ContainerAttach(context.TODO(), b.containerID, types.ContainerAttachOptions{
		Stream: true,
		Stdout: true,
		Stderr: true,
	})
	if err != nil {
		return errors.WithMessage(err, "attaching to container")
	}
	go func() {
		// TODO: Close as appropriate
		for true {
			_, _ = response.Reader.WriteTo(standardLog)
		}
	}()
	return nil
}

func (b *buildandrun) Stop(workingDir string, getenv func(string) string) ([]byte, error) {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	duration := time.Duration(0)
	err := b.client.ContainerStop(context.TODO(), b.containerID, &duration)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if !b.Backend.Persistent {
		err = b.client.ContainerRemove(context.TODO(), b.containerID, types.ContainerRemoveOptions{})
		if err != nil {
			return nil, errors.WithStack(err)
		}
	}

	close(b.done)
	return nil, nil
}

func (b *buildandrun) Status() (services.BackendStatus, error) {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	if b.client == nil || b.containerID == "" {
		return services.BackendStatus{}, nil
	}

	container, err := b.client.ContainerInspect(context.TODO(), b.containerID)
	if err != nil {
		errors.WithMessage(err, "pulling image")
	}

	var ports []string
	if container.HostConfig != nil {
		for _, bindings := range container.HostConfig.PortBindings {
			for _, binding := range bindings {
				ports = append(ports, binding.HostPort)
			}
		}
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
	// TODO: Pipe to output
	_, err := b.client.ImagePull(context.TODO(), b.Backend.Image, types.ImagePullOptions{
		All: true,
	})
	if err != nil {
		errors.WithMessage(err, "pulling image")
	}

	imgs, err := b.client.ImageList(context.TODO(), types.ImageListOptions{})
	if err != nil {
		return "", errors.WithStack(err)
	}
	var imgID string
	for _, img := range imgs {
		if len(img.RepoTags) > 0 && strings.Contains(img.RepoTags[0], b.Backend.Image) {
			imgID = img.ID
		}
	}
	return imgID, nil
}

func (b *buildandrun) findContainer() (string, bool, error) {
	var containerID string
	containers, err := b.client.ContainerList(context.TODO(), types.ContainerListOptions{
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
