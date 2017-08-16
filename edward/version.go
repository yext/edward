package edward

import "github.com/yext/edward/common"

func (c *Client) Version() string {
	return common.EdwardVersion
}
