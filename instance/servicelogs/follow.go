package servicelogs

import (
	"log"
	"os"
	"time"

	"github.com/hpcloud/tail"
	"github.com/pkg/errors"
)

type LogFollower struct {
	logs chan LogLine
	done chan struct{}

	runLog string
}

// NewLogFollower creates a log follower that tails a log file for the specified service
func NewLogFollower(runLog string) *LogFollower {
	return &LogFollower{
		runLog: runLog,
		done:   make(chan struct{}),
	}
}

func (f *LogFollower) Start() <-chan LogLine {
	logs := make(chan LogLine)
	go f.doStart(logs)
	return logs
}

func (f *LogFollower) doStart(logs chan<- LogLine) {
	// Wait for file to exist
	var exists bool
	for !exists {
		_, err := os.Stat(f.runLog)
		exists = !os.IsNotExist(err)
		select {
		case <-f.done:
			close(logs)
			return
		default:
			time.Sleep(time.Millisecond * 100)
		}
	}

	err := doFollowServiceLog(f.runLog, 0, logs, f.done)
	if err != nil {
		log.Print("error", err)
		return
	}
}

func (f *LogFollower) Stop() {
	close(f.done)
}

func doFollowServiceLog(file string, skipLines int, logChannel chan<- LogLine, done <-chan struct{}) error {
	t, err := tail.TailFile(file, tail.Config{
		Follow: true,
		Logger: tail.DiscardingLogger,
		Location: &tail.SeekInfo{
			Offset: 0,
			Whence: 0,
		},
	})
	if err != nil {
		return errors.WithStack(err)
	}
	defer func() {
		close(logChannel)
	}()
	var linesSkipped int
	for line := range t.Lines {
		if linesSkipped < skipLines {
			linesSkipped++
			continue
		}
		lineData, err := ParseLogLine(line.Text)
		if err != nil {
			t.Err()
		}
		logChannel <- lineData

		select {
		case <-done:
			return nil
		default:
		}
	}
	return nil
}
