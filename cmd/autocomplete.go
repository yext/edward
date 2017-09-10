package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/yext/edward/common"
	"github.com/yext/edward/config"
)

func autocompleteServicesAndGroups(logger *log.Logger) {
	printCommandChildren(RootCmd)

	configPath, err := config.GetConfigPathFromWorkingDirectory()
	if err != nil {
		logger.Println("Autocomplete> Error getting config path:", err)
		return
	}
	if configPath == "" {
		logger.Println("Autocomplete> Config file not found")
		return
	}
	cfg, err := config.LoadConfig(configPath, common.EdwardVersion, logger)
	if err != nil {
		logger.Println("Autocomplete> Error loading config:", err)
		return
	}

	var names []string
	for _, service := range cfg.ServiceMap {
		names = append(names, service.Name)
		names = append(names, service.Aliases...)
	}
	for _, group := range cfg.GroupMap {
		names = append(names, group.Name)
		names = append(names, group.Aliases...)
	}

	for _, name := range names {
		fmt.Println(name)
	}
}

func printCommandChildren(cmd *cobra.Command) {
	for _, cmd := range cmd.Commands() {
		fmt.Println(cmd.Use)
		printCommandChildren(cmd)
	}
}
