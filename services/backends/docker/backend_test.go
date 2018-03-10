package docker_test

import (
	"encoding/json"
	"testing"

	"github.com/theothertomelliott/must"
	"github.com/yext/edward/services/backends/docker"
)

func TestUnmarshal(t *testing.T) {
	var tests = []struct {
		name            string
		json            []byte
		expectedBackend docker.Backend
	}{
		{
			name: "explicit ports",
			json: []byte(`
				{
					"image": "kitematic/hello-world-nginx",
					"containerConfig": {
						"exposedPorts": {
							"8080/tcp": {}
						}
					},
					"hostConfig": {
					"portBindings": {
						"8080/tcp": [
								{"HostPort": "8081/tcp"}
							]
						}
					}
				}`,
			),
			expectedBackend: basicBackend,
		},
		{
			name: "convenience ports",
			json: []byte(`
				{
					"image": "kitematic/hello-world-nginx",
					"ports": ["8080:8081"]
				}`,
			),
			expectedBackend: basicWithConvenience,
		},
		{
			name: "convenience ports - explicit protocol",
			json: []byte(`
				{
					"image": "kitematic/hello-world-nginx",
					"ports": ["8080/tcp:8081/tcp"]
				}`,
			),
			expectedBackend: basicWithConvenienceExplicitProtocol,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var out docker.Backend
			err := json.Unmarshal(test.json, &out)
			if err != nil {
				t.Error(err)
			}
			must.BeEqual(t, test.expectedBackend, out, "backend was not as expected")
		})
	}
}

func TestUnmarshalErrors(t *testing.T) {
	var tests = []struct {
		name string
		json []byte
	}{
		{
			name: "invalid mapping strings",
			json: []byte(`
				{
					"ports": ["111111111"]
				}`,
			),
		},
		{
			name: "not a numeric port",
			json: []byte(`
				{
					"ports": ["string:8081"]
				}`,
			),
		},
		{
			name: "not a numeric port (host)",
			json: []byte(`
				{
					"ports": ["8080:string"]
				}`,
			),
		},
		{
			name: "invalid protocol",
			json: []byte(`
				{
					"image": "kitematic/hello-world-nginx",
					"ports": ["8080/abc:8081/tcp"]
				}`,
			),
		},
		{
			name: "invalid protocol (host)",
			json: []byte(`
				{
					"image": "kitematic/hello-world-nginx",
					"ports": ["8080/tcp:8081/abc"]
				}`,
			),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var out docker.Backend
			err := json.Unmarshal(test.json, &out)
			if err == nil {
				t.Error("expected an error")
			}
		})
	}
}

func TestMarshal(t *testing.T) {
	var tests = []struct {
		name    string
		backend docker.Backend
	}{
		{
			name:    "explicit ports",
			backend: basicBackend,
		},
		{
			name:    "convenience ports",
			backend: basicWithConvenience,
		},
		{
			name:    "convenience ports - explicit protocol",
			backend: basicWithConvenienceExplicitProtocol,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data, err := json.Marshal(test.backend)
			if err != nil {
				t.Error(err)
			}
			var out docker.Backend
			err = json.Unmarshal(data, &out)
			if err != nil {
				t.Error(err)
			}
			must.BeEqual(t, test.backend, out, "backend was not as expected")
		})
	}
}

var basicBackend = docker.Backend{
	Image: "kitematic/hello-world-nginx",
	ContainerConfig: docker.Config{
		ExposedPorts: map[docker.Port]struct{}{
			docker.Port("8080/tcp"): struct{}{},
		},
	},
	HostConfig: docker.HostConfig{
		PortBindings: map[docker.Port][]docker.PortBinding{
			docker.Port("8080/tcp"): []docker.PortBinding{
				{
					HostPort: "8081/tcp",
				},
			},
		},
	},
}

var basicWithConvenience = docker.Backend{
	Image: "kitematic/hello-world-nginx",
	Ports: []*docker.PortMapping{
		{
			ContainerPort: docker.Port("8080/tcp"),
			HostPort:      docker.Port("8081/tcp"),
			Original:      "8080:8081",
		},
	},
	ContainerConfig: docker.Config{
		ExposedPorts: map[docker.Port]struct{}{
			docker.Port("8080/tcp"): struct{}{},
		},
	},
	HostConfig: docker.HostConfig{
		PortBindings: map[docker.Port][]docker.PortBinding{
			docker.Port("8080/tcp"): []docker.PortBinding{
				{
					HostPort: "8081/tcp",
				},
			},
		},
	},
}

var basicWithConvenienceExplicitProtocol = docker.Backend{
	Image: "kitematic/hello-world-nginx",
	Ports: []*docker.PortMapping{
		{
			ContainerPort: docker.Port("8080/tcp"),
			HostPort:      docker.Port("8081/tcp"),
			Original:      "8080/tcp:8081/tcp",
		},
	},
	ContainerConfig: docker.Config{
		ExposedPorts: map[docker.Port]struct{}{
			docker.Port("8080/tcp"): struct{}{},
		},
	},
	HostConfig: docker.HostConfig{
		PortBindings: map[docker.Port][]docker.PortBinding{
			docker.Port("8080/tcp"): []docker.PortBinding{
				{
					HostPort: "8081/tcp",
				},
			},
		},
	},
}
