// +build disabled

package docker

import (
	"os"
	"testing"

	docker "github.com/fsouza/go-dockerclient"
)

func TestBuild(t *testing.T) {
	client, err := docker.NewClientFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	err = client.BuildImage(docker.BuildImageOptions{
		Name:         "test",
		Dockerfile:   "./Dockerfile",
		OutputStream: os.Stdout,
		ContextDir:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
}
