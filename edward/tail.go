package edward

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/hpcloud/tail"
	"github.com/pkg/errors"
	"github.com/yext/edward/runner"
	"github.com/yext/edward/services"
)

const (
	displayLine    = iota
	displaySummary = iota
	displayPretty  = iota
)

//var displayMode = displayLine
var displayMode = displayPretty

type byTime []runner.LogLine

func (a byTime) Len() int           { return len(a) }
func (a byTime) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byTime) Less(i, j int) bool { return a[i].Time.Before(a[j].Time) }

func printMessage(logMessage runner.LogLine, multiple bool) {

	colored := false
	message := strings.TrimSpace(logMessage.Message)

	if len(message) == 0 {
		return
	}

	if multiple {
		print("[")
		color.Set(color.FgHiYellow)
		print(logMessage.Name)
		if logMessage.Stream == "messages" {
			print(" (edward)")
		}
		color.Unset()
		print("]: ")
	}

	if logMessage.Stream == "stderr" {
		//color.Set(color.FgRed)
		color.Set(color.FgWhite)
	}
	if logMessage.Stream == "messages" {
		color.Set(color.FgYellow)
	}

	if strings.Contains(message, `"level":"error"`) {
		color.Set(color.FgRed)
		colored = true
	}

	// if message is json, then purty print it
	switch displayMode {
	case displayLine: // nothing spesh
	case displaySummary: // just the handler
		if strings.HasPrefix(message, "{") && strings.HasSuffix(message, "}") {
			if i := strings.Index(message, `"handler":"`); i != -1 {
				if i2 := strings.Index(message[i+11:], `"`); i2 != -i {
					message = message[i+11 : i+11+i2-1]
				}
			}
		}
	case displayPretty: // purdy print the json
		if strings.HasPrefix(message, "{") && strings.HasSuffix(message, "}") {
			var data map[string]interface{}
			json.Unmarshal([]byte(message), &data)
			p, err := json.MarshalIndent(data, "", "  ")
			if err == nil {
				if len(data) > 0 {
					message = string(p)
				} else {
					color.Set(color.FgHiMagenta)
					fmt.Println() // put the purple stuff on its own line
					colored = true
				}
			}
		} else {
			color.Set(color.FgGreen)
			colored = true
		}
	}

	if !colored && strings.Index(message, `"handler":`) == -1 {
		color.Set(color.FgCyan)
	}
	fmt.Printf("%v\n", strings.TrimSpace(message))
	color.Unset()
}

func followGroupLog(logDir string, group *services.ServiceGroupConfig, logChannel chan runner.LogLine) ([]runner.LogLine, error) {
	var lines []runner.LogLine
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

func followServiceLog(logDir string, service *services.ServiceConfig, logChannel chan runner.LogLine) ([]runner.LogLine, error) {
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
	var initialLines []runner.LogLine
	// create a new scanner and read the file line by line
	scanner := bufio.NewScanner(logFile)
	var lineCount int
	for scanner.Scan() {
		text := scanner.Text()
		lineCount++
		var line runner.LogLine
		line, err = runner.ParseLogLine(text)
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

func doFollowServiceLog(logDir string, service *services.ServiceConfig, skipLines int, logChannel chan runner.LogLine) error {
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
		lineData, err := runner.ParseLogLine(line.Text)
		if err != nil {
			return errors.WithStack(err)
		}
		logChannel <- lineData
	}
	return nil
}
