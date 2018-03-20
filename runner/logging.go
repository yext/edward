package runner

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// Logger provides a simple interface for logging
type Logger interface {
	Printf(format string, a ...interface{})
}

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

// Log provides the io.Writer interface to publish service logs to file
type Log struct {
	file   *os.File
	name   string
	stream string
	lines  int
}

func (r *Log) Len() int {
	return r.lines
}

// Printf prints a message to a RunnerLog
func (r *Log) Printf(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	r.Write([]byte(msg))
}

// Write writes a slice of bytes to a RunnerLog
func (r *Log) Write(p []byte) (int, error) {
	r.lines++
	fmt.Println(strings.TrimRight(string(p), "\n"))
	lineData := LogLine{
		Name:    r.name,
		Time:    time.Now(),
		Stream:  r.stream,
		Message: strings.TrimSpace(string(p)),
	}

	jsonContent, err := json.Marshal(lineData)
	if err != nil {
		return 0, errors.Wrap(err, "could not prepare log line")
	}

	line := fmt.Sprintln(string(jsonContent))
	count, err := r.file.Write([]byte(line))
	if err != nil {
		return count, errors.Wrap(err, "could not write log line")
	}
	return len(p), nil
}
