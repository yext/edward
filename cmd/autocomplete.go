package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/yext/edward/common"
	"github.com/yext/edward/config"
)

func autocompleteServicesAndGroups(logger *log.Logger) {
	printCommandChildren(RootCmd)

	wd, err := os.Getwd()
	if err != nil {
		logger.Println("Autocomplete> Error getting working dir:", err)
		return
	}

	err = config.LoadSharedConfig(getConfigPath(wd), common.EdwardVersion, logger)
	if err != nil {
		logger.Println("Autocomplete> Error loading config:", err)
	}
	names := append(config.GetAllGroupNames(), config.GetAllServiceNames()...)
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
