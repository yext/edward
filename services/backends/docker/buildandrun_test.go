package docker_test

import (
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/yext/edward/services"
	"github.com/yext/edward/services/backends/docker"
)

func TestMain(m *testing.M) {
	// Enable docker tests with a flag
	dockerEnabled := flag.Bool("edward.docker", false, "Enable docker tests for Edward.")
	flag.Parse()
	if dockerEnabled == nil || !*dockerEnabled {
		log.Println("Docker tests disabled")
		return
	}

	// Register necessary backends
	services.RegisterBackend(&docker.Loader{})
	os.Exit(m.Run())
}

func TestBuild(t *testing.T) {
	service := &services.ServiceConfig{
		Name: "testservice",
		Path: func(in string) *string {
			return &in
		}("testdata/simple"),
		Backends: []*services.BackendConfig{
			{
				Type: "docker",
				Config: &docker.Backend{
					Build: ".",
					ContainerConfig: docker.Config{
						ExposedPorts: map[docker.Port]struct{}{
							"8080/tcp": struct{}{},
						},
					},
					HostConfig: docker.HostConfig{
						PortBindings: map[docker.Port][]docker.PortBinding{
							"80/tcp": []docker.PortBinding{
								{
									HostPort: "51432/tcp",
								},
							},
						},
					},
				},
			},
		},
	}
	b, err := services.GetBuilder(services.OperationConfig{}, service)
	if err != nil {
		t.Error(err)
		return
	}

	out, err := b.Build("", nil)
	t.Log(string(out))
	if err != nil {
		t.Error(err)
		return
	}

	doStartTest(t, service)
}

func TestBuildAltFile(t *testing.T) {
	service := &services.ServiceConfig{
		Name: "testservice",
		Path: func(in string) *string {
			return &in
		}("testdata/alternate_filename"),
		Backends: []*services.BackendConfig{
			{
				Type: "docker",
				Config: &docker.Backend{
					Build: "AltDocker",
					ContainerConfig: docker.Config{
						ExposedPorts: map[docker.Port]struct{}{
							"8080/tcp": struct{}{},
						},
					},
					HostConfig: docker.HostConfig{
						PortBindings: map[docker.Port][]docker.PortBinding{
							"80/tcp": []docker.PortBinding{
								{
									HostPort: "51432/tcp",
								},
							},
						},
					},
				},
			},
		},
	}
	b, err := services.GetBuilder(services.OperationConfig{}, service)
	if err != nil {
		t.Error(err)
		return
	}

	out, err := b.Build("", nil)
	t.Log(string(out))
	if err != nil {
		t.Error(err)
		return
	}

	doStartTest(t, service)
}

func TestStart(t *testing.T) {
	service := &services.ServiceConfig{
		Name: "testservice",
		Backends: []*services.BackendConfig{
			{
				Type: "docker",
				Config: &docker.Backend{
					Image: "kitematic/hello-world-nginx:latest",
					ContainerConfig: docker.Config{
						ExposedPorts: map[docker.Port]struct{}{
							"8080/tcp": struct{}{},
						},
					},
					HostConfig: docker.HostConfig{
						PortBindings: map[docker.Port][]docker.PortBinding{
							"80/tcp": []docker.PortBinding{
								{
									HostPort: "51432/tcp",
								},
							},
						},
					},
				},
			},
		},
	}
	doStartTest(t, service)
}

func doStartTest(t *testing.T, service *services.ServiceConfig) {
	b, err := services.GetRunner(services.OperationConfig{}, service)
	if err != nil {
		t.Error(err)
		return
	}

	err = b.Start(os.Stdout, os.Stderr)
	if err != nil {
		t.Error(err)
		return
	}
	defer func() {
		_, err = b.Stop("", nil)
		if err != nil {
			t.Error(err)
		}

		_, err := http.Get("http://localhost:51432/")
		if err == nil {
			t.Error("Did not expect request to stopped container to succeed")
		}
	}()

	resp, err := http.Get("http://localhost:51432/")
	if err != nil {
		t.Error(err)
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Error(err)
		return
	}
	if !strings.Contains(string(body), "nginx container") {
		t.Errorf("Response was not as expected:\n%s", string(body))
	}
}
