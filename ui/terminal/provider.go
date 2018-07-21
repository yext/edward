package terminal

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/yext/edward/services"
	"github.com/yext/edward/ui"
)

var _ ui.Provider = &Provider{}

type Provider struct {
}

func (p *Provider) Infof(format string, args ...interface{}) {
	fmt.Printf(format, args...)
	fmt.Println()
}

func (p *Provider) Errorf(format string, args ...interface{}) {
	fmt.Printf(format, args...)
	fmt.Println()
}

func (p *Provider) Confirm(format string, args ...interface{}) bool {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf(format, args...)
		fmt.Print(" [y/n]?")

		response, err := reader.ReadString('\n')
		if err != nil {
			return false
		}

		response = strings.ToLower(strings.TrimSpace(response))

		if response == "y" || response == "yes" {
			return true
		} else if response == "n" || response == "no" {
			return false
		}
	}
}

func (p *Provider) List(services []services.ServiceOrGroup, groups []services.ServiceOrGroup) {
	p.Infof("Services and groups")
	p.Infof("Groups:")
	for _, g := range groups {
		if g.GetDescription() != "" {
			p.Infof("\t%v: %v", g.GetName(), g.GetDescription())
		} else {
			p.Infof("\t%v", g.GetName())
		}
	}
	p.Infof("Services:")
	for _, s := range services {
		if s.GetDescription() != "" {
			p.Infof("\t%v: %v", s.GetName(), s.GetDescription())
		} else {
			p.Infof("\t%v", s.GetName())
		}
	}
}
