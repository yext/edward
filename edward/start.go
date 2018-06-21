package edward

import (
	"log"

	"github.com/pkg/errors"
	"github.com/yext/edward/services"
)

func (c *Client) Start(names []string, skipBuild bool, noWatch bool, exclude []string) error {
	log.Println("Start:", names, skipBuild, noWatch, exclude)
	if len(names) == 0 {
		if len(c.Backends) == 0 {
			return errors.New("At least one service or group must be specified")
		}
	}

	sgs, err := c.getServicesOrGroups(names)
	if err != nil {
		return errors.WithStack(err)
	}
	if startingServicesOnly(sgs) && len(c.Backends) != 0 {
		names = append(getServiceNamesFromBackends(c), names...)
		sgs, err = c.getServicesOrGroups(names)
		if err != nil {
			return errors.WithStack(err)
		}
	}
	if c.ServiceChecks != nil {
		err = c.ServiceChecks(sgs)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	err = c.startAndTrack(sgs, skipBuild, noWatch, exclude, c.EdwardExecutable)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func startingServicesOnly(sgs []services.ServiceOrGroup) bool {
	return services.CountServices(sgs) == len(sgs)
}

func getServiceNamesFromBackends(c *Client) []string {
	names := make([]string, 0, len(c.Backends))
	for service := range c.Backends {
		names = append(names, service)
	}
	return names
}
