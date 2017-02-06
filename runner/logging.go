package runner

import (
	"encoding/json"
	"time"

	"github.com/pkg/errors"
)

type LogLine struct {
	Name    string
	Time    time.Time
	Stream  string
	Message string
}

func ParseLogLine(line string) (LogLine, error) {
	var lineData LogLine
	err := json.Unmarshal([]byte(line), &lineData)
	if err != nil {
		return LogLine{}, errors.WithStack(err)
	}
	return lineData, nil
}
