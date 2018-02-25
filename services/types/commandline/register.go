package commandline

import "github.com/yext/edward/services"

// TypeCommandLine identifies a service as being built and launched via the command line.
// Defined in this package as a default
const TypeCommandLine services.Type = "commandline"

func init() {
	services.RegisterServiceType(TypeCommandLine, &CommandLineLoader{})
	services.SetDefaultServiceType(TypeCommandLine)
}
