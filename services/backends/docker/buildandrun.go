package docker

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
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

	hijackedResponse *types.HijackedResponse

	mtx sync.Mutex
}

var _ services.Builder = &buildandrun{}
var _ services.Runner = &buildandrun{}

func (b *buildandrun) Build(workingDir string, getenv func(string) string, output io.Writer) error {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	if b.Backend.Dockerfile != "" {
		r, w := io.Pipe()
		servicePath := workingDir
		if b.Service.Path != nil {
			servicePath = path.Join(servicePath, *b.Service.Path)
		}

		go func() {
			err := tarDir(servicePath, w)
			if err != nil {
				fmt.Println(err)
			}
			w.Close()
		}()

		relativeBuild := b.Backend.Dockerfile
		buildPath := path.Join(servicePath, relativeBuild)
		fi, err := os.Stat(buildPath)
		if err != nil {
			return errors.WithStack(err)
		}
		if fi.IsDir() {
			relativeBuild = path.Join(relativeBuild, "Dockerfile")
		}
		tag := b.imageTag()
		response, err := b.client.ImageBuild(context.Background(), r, types.ImageBuildOptions{
			Tags: []string{
				tag,
			},
			Dockerfile: relativeBuild,
		})
		if err != nil {
			return errors.WithStack(err)
		}

		err = jsonmessage.DisplayJSONMessagesStream(response.Body, output, 0, false, nil)
		if err != nil {
			return errors.WithMessage(err, "reading messages")
		}
		return nil
	}
	return nil
}

func (b *buildandrun) Start(standardLog io.Writer, errorLog io.Writer) error {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	b.done = make(chan struct{})

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
		var networkConfig network.NetworkingConfig

		struct2struct.Marshal(&backendConfig, &config)
		struct2struct.Marshal(&b.Backend.HostConfig, &hostConfig)
		struct2struct.Marshal(&b.Backend.NetworkConfig, &networkConfig)

		config.Image = imgID
		container, err := b.client.ContainerCreate(
			context.Background(),
			&config,
			&hostConfig,
			&networkConfig,
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

	err = b.client.ContainerStart(context.Background(), b.containerID, types.ContainerStartOptions{})
	if err != nil {
		return errors.WithMessage(err, "starting container")
	}

	response, err := b.client.ContainerAttach(context.Background(), b.containerID, types.ContainerAttachOptions{
		Stream: true,
		Stdout: true,
		Stderr: true,
	})
	if err != nil {
		return errors.WithMessage(err, "attaching to container")
	}
	b.hijackedResponse = &response
	go func() {
		if b.hijackedResponse != nil {
			_, _ = io.Copy(standardLog, b.hijackedResponse.Reader)
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

	if b.hijackedResponse != nil {
		b.hijackedResponse.Close()
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
	if b.Backend.ContainerName != "" {
		return b.Backend.ContainerName
	}
	return fmt.Sprintf("edward-%s", b.Service.Name)
}

func (b *buildandrun) imageTag() string {
	if b.Backend.Dockerfile != "" {
		return fmt.Sprintf(
			"edward/%s",
			strings.ToLower(
				b.Service.IdentifyingFilenameWithEncoding(
					base64.RawURLEncoding,
				),
			),
		)
	}
	return b.Backend.Image
}

func (b *buildandrun) findImage(standardLog io.Writer) (string, error) {
	ctx := context.Background()
	imgID, err := b.getImageId(ctx)
	if err != nil {
		return "", errors.WithStack(err)
	}
	if imgID != "" {
		return imgID, nil
	}
	if b.Backend.Image != "" {
		output, err := b.client.ImagePull(ctx, b.Backend.Image, types.ImagePullOptions{
			All: true,
		})
		if err != nil {
			return "", errors.WithMessage(err, "pulling image")
		}
		err = jsonmessage.DisplayJSONMessagesStream(output, standardLog, 0, false, nil)
		if err != nil {
			return "", errors.WithMessage(err, "reading messages")
		}
		output.Close()
	}
	imgID, err = b.getImageId(ctx)
	if err != nil {
		return "", errors.WithStack(err)
	}
	return imgID, nil
}

func (b *buildandrun) getImageId(ctx context.Context) (string, error) {
	imgs, err := b.client.ImageList(ctx, types.ImageListOptions{})
	if err != nil {
		return "", errors.WithStack(err)
	}
	for _, img := range imgs {
		if len(img.RepoTags) > 0 && strings.HasPrefix(img.RepoTags[0], b.imageTag()) {
			return img.ID, nil
		}
	}
	return "", nil
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
