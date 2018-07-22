package terminal

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/yext/edward/runner"
)

func (p *Provider) ShowLog(logs <-chan runner.LogLine, multiple bool) {
	go func() {
		for log := range logs {
			printMessage(log, multiple)
		}
	}()
}

func printMessage(logMessage runner.LogLine, multiple bool) {

	message := strings.TrimSpace(logMessage.Message)

	if len(message) == 0 {
		return
	}

	if multiple {
		print("[")
		color.Set(color.FgHiYellow)
		print(logMessage.Name)
		if logMessage.Stream == "messages" {
			print(" (edward)")
		}
		color.Unset()
		print("]: ")
	}

	if logMessage.Stream == "stderr" {
		color.Set(color.FgRed)
	}
	if logMessage.Stream == "messages" {
		color.Set(color.FgYellow)
	}

	fmt.Printf("%v\n", strings.TrimSpace(message))
	color.Unset()
}
