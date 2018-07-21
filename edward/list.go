package edward

func (c *Client) List() error {
	groups := c.getAllGroupsSorted()
	services := c.getAllServicesSorted()

	c.UI.List(services, groups)

	return nil
}
