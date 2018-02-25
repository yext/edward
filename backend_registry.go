package main

import (
	"github.com/yext/edward/services"
	"github.com/yext/edward/services/backends/commandline"
)

// RegisterBackends configures all supported service backends.
func RegisterBackends() {
	services.RegisterBackend(commandline.TypeCommandLine, &commandline.CommandLineLoader{})
	services.SetDefaultBackend(commandline.TypeCommandLine)
}
