package edward

import (
	"fmt"
)

func (c *Client) List() error {
	groups := c.getAllGroupsSorted()
	services := c.getAllServicesSorted()

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
