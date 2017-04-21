package main

import (
	"fmt"

	"github.com/urfave/cli"
	"github.com/yext/edward/config"
)

// didAutoComplete indicates whether or not autocompletion was called
var didAutoComplete bool

func autocompleteServicesAndGroups(c *cli.Context) {
	didAutoComplete = true
	config.LoadSharedConfig(getConfigPath(), edwardVersion, logger)
	names := append(config.GetAllGroupNames(), config.GetAllServiceNames()...)
	for _, name := range names {
		fmt.Println(name)
	}
}
