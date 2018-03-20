package commandline

import (
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/yext/edward/services"
)

type LegacyUnmarshaler struct{}

func (l *LegacyUnmarshaler) Unmarshal(data []byte, c *services.ServiceConfig) error {
	var backend Backend
	var empty Backend
	err := json.Unmarshal(data, &backend)
	if err != nil {
		return errors.WithStack(err)
	}
	if backend != empty {
		c.Backends = append(c.Backends, &services.BackendConfig{
			Type:   "commandline",
			Config: &backend,
		})
	}

	return nil
}
