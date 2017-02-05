package runner

import (
	"encoding/json"
	"time"

	"github.com/pkg/errors"
	"github.com/yext/edward/services"
)

type LogLine struct {
	Name    string
	Time    time.Time
	Stream  string
	Message string
}

func ParseLogLine(line string, service *services.ServiceConfig) (LogLine, error) {
	var lineData LogLine
	err := json.Unmarshal([]byte(line), &lineData)
	if err != nil {
		return LogLine{}, errors.WithStack(err)
	}
	if service != nil {
		lineData.Name = service.Name
	}
	return lineData, nil
}
