package edward

import (
	"bufio"
	"os"

	"github.com/hpcloud/tail"
	"github.com/pkg/errors"
	"github.com/yext/edward/instance/servicelogs"
	"github.com/yext/edward/services"
)

type byTime []servicelogs.LogLine

func (a byTime) Len() int           { return len(a) }
func (a byTime) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byTime) Less(i, j int) bool { return a[i].Time.Before(a[j].Time) }

func followGroupLog(logDir string, group *services.ServiceGroupConfig, logChannel chan servicelogs.LogLine) ([]servicelogs.LogLine, error) {
	var lines []servicelogs.LogLine
	for _, group := range group.Groups {
		newLines, err := followGroupLog(logDir, group, logChannel)
		lines = append(lines, newLines...)
		if err != nil {
			return nil, err
		}
	}
	for _, service := range group.Services {
		newLines, err := followServiceLog(logDir, service, logChannel)
		lines = append(lines, newLines...)
		if err != nil {
			return nil, err
		}
	}
	return lines, nil
}

func followServiceLog(logDir string, service *services.ServiceConfig, logChannel chan servicelogs.LogLine) ([]servicelogs.LogLine, error) {
	// Skip services that don't include a launch step
	if !service.Backend().HasLaunchStep() {
		return nil, nil
	}

	runLog := service.GetRunLog(logDir)
	logFile, err := os.Open(runLog)
	defer logFile.Close()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	var initialLines []servicelogs.LogLine
	// create a new scanner and read the file line by line
	scanner := bufio.NewScanner(logFile)
	var lineCount int
	for scanner.Scan() {
		text := scanner.Text()
		lineCount++
		var line servicelogs.LogLine
		line, err = servicelogs.ParseLogLine(text)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		initialLines = append(initialLines, line)
	}

	// check for errors
	if err = scanner.Err(); err != nil {
		return nil, errors.WithStack(err)
	}

	go doFollowServiceLog(logDir, service, lineCount, logChannel)
	return initialLines, nil
}

func doFollowServiceLog(logDir string, service *services.ServiceConfig, skipLines int, logChannel chan servicelogs.LogLine) error {
	runLog := service.GetRunLog(logDir)
	t, err := tail.TailFile(runLog, tail.Config{
		Follow: true,
		Logger: tail.DiscardingLogger,
	})
	if err != nil {
		return errors.WithStack(err)
	}
	var linesSkipped int
	for line := range t.Lines {
		if linesSkipped < skipLines {
			linesSkipped++
			continue
		}
		lineData, err := servicelogs.ParseLogLine(line.Text)
		if err != nil {
			return errors.WithStack(err)
		}
		logChannel <- lineData
	}
	return nil
}
