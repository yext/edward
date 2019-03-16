package services

import (
	"encoding/json"
	"time"

	"github.com/pkg/errors"
)

type Duration struct {
	time.Duration
}

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var (
		v   interface{}
		err error
	)
	if err = json.Unmarshal(b, &v); err != nil {
		return errors.WithStack(err)
	}
	switch value := v.(type) {
	case string:
		d.Duration, err = time.ParseDuration(value)
		if err != nil {
			return errors.WithStack(err)
		}
		return nil
	default:
		return errors.New("invalid duration")
	}
}
