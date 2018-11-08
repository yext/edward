package runner

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/yext/edward/instance/servicelogs"
)

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
	lineData := servicelogs.LogLine{
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
