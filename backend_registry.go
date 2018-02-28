package main

import (
	"github.com/yext/edward/services"
	"github.com/yext/edward/services/backends/commandline"
	"github.com/yext/edward/services/backends/docker"
)

// RegisterBackends configures all supported service backends.
func RegisterBackends() {
	services.RegisterDefaultBackend(&commandline.CommandLineLoader{})
	services.RegisterBackend(&docker.Loader{})
}
