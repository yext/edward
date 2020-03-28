package edward

import (
	"path/filepath"
	"sort"

	"github.com/pkg/errors"
	"github.com/yext/edward/config"
	"github.com/yext/edward/services"
)

// LoadConfig loads an Edward project config into the shared maps for this client
func (c *Client) LoadConfig(edwardVersion string) error {
	if c.Config != "" {
		c.basePath = filepath.Dir(c.Config)
		cfg, err := config.LoadConfig(c.Config, edwardVersion)
		if err != nil {
			return errors.WithMessage(err, c.Config)
		}

		c.telemetryScript = cfg.TelemetryScript
		c.serviceMap = cfg.ServiceMap
		c.groupMap = cfg.GroupMap
		return nil
	}

	return errors.New("No config file found")
}

// getServicesOrGroups returns services and groups matching any of the provided names
func (c *Client) getServicesOrGroups(names []string) ([]services.ServiceOrGroup, error) {
	var outSG []services.ServiceOrGroup
	for _, name := range names {
		sg, err := c.getServiceOrGroup(name)
		if err != nil {
			return nil, err
		}
		outSG = append(outSG, sg)
	}
	return outSG, nil
}

// getServiceOrGroup returns the service/group matching the provided name
func (c *Client) getServiceOrGroup(name string) (services.ServiceOrGroup, error) {
	if group, ok := c.groupMap[name]; ok {
		return group, nil
	}
	if service, ok := c.serviceMap[name]; ok {
		return service, nil
	}
	// Check aliases
	for _, group := range c.groupMap {
		if group.Matches(name) {
			return group, nil
		}
	}
	for _, service := range c.serviceMap {
		if service.Matches(name) {
			return service, nil
		}
	}
	return nil, errors.New("Service or group not found")
}

// getAllGroupsSorted returns a slice of all groups, sorted by name
func (c *Client) getAllGroupsSorted() []services.ServiceOrGroup {
	var as []services.ServiceOrGroup
	for _, group := range c.groupMap {
		as = append(as, group)
	}
	sort.Sort(serviceOrGroupByName(as))
	return as
}

// getAllServicesSorted returns a slice of all services, sorted by name
func (c *Client) getAllServicesSorted() []services.ServiceOrGroup {
	var as []services.ServiceOrGroup
	for _, service := range c.serviceMap {
		as = append(as, service)
	}
	sort.Sort(serviceOrGroupByName(as))
	return as
}

type serviceOrGroupByName []services.ServiceOrGroup

func (s serviceOrGroupByName) Len() int {
	return len(s)
}
func (s serviceOrGroupByName) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s serviceOrGroupByName) Less(i, j int) bool {
	return s[i].GetName() < s[j].GetName()
}
