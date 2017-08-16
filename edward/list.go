package edward

import (
	"fmt"

	"github.com/yext/edward/config"
)

func (c *Client) List() error {
	groups := config.GetAllGroupsSorted()
	services := config.GetAllServicesSorted()

	fmt.Fprintln(c.Output, "Services and groups")
	fmt.Fprintln(c.Output, "Groups:")
	for _, g := range groups {
		fmt.Fprintln(c.Output, "\t", g.GetName(), ": ", g.GetDescription())
	}
	fmt.Fprintln(c.Output, "Services:")
	for _, s := range services {
		fmt.Fprintln(c.Output, "\t", s.GetName(), ": ", s.GetDescription())
	}

	return nil
}
