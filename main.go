package main

import (
	"github.com/yext/edward/cmd"
)

func main() {
	// Initialization
	RegisterBackends()

	cmd.Execute()
}
