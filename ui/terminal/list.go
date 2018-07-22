package terminal

import "github.com/yext/edward/services"

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
