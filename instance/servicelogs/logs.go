package servicelogs

import (
	"encoding/json"
	"time"

	"github.com/pkg/errors"
)

// LogLine represents a line in an Edward service log
type LogLine struct {
	Name    string
	Time    time.Time
	Stream  string
	Message string
}

// ParseLogLine parses the JSON representation of a log line into a LogLine
func ParseLogLine(line string) (LogLine, error) {
	var lineData LogLine
	err := json.Unmarshal([]byte(line), &lineData)
	if err != nil {
		return LogLine{}, errors.WithStack(err)
	}
	return lineData, nil
}
