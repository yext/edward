package edward

import (
	"fmt"
	"io"
	"os"

	"github.com/pkg/errors"
	"github.com/yext/edward/instance"
	"github.com/yext/edward/instance/processes"
	"github.com/yext/edward/instance/servicelogs"
	"github.com/yext/edward/services"
)

// TipLog outputs the last few log lines for each service
func (c *Client) TipLog(names []string, lineCount int) error {
	sgs, err := c.getServiceList(names, false)
	if err != nil {
		return errors.WithStack(err)
	}

	c.tipLogServicesOrGroups(sgs, lineCount)

	return nil
}

func (c *Client) tipLogServicesOrGroups(sgs []services.ServiceOrGroup, lineCount int) error {
	for _, sg := range sgs {
		switch v := sg.(type) {
		case *services.ServiceConfig:
			i, err := instance.Load(c.DirConfig, &processes.Processes{}, v, services.ContextOverride{})
			if err != nil {
				return errors.WithStack(err)
			}
			if i.Pid == 0 {
				continue
			}
			lines, err := getLastLinesOfLog(c.DirConfig.LogDir, v, lineCount)
			if err != nil {
				return errors.WithStack(err)
			}
			c.UI.Infof("==== %v ====", v.GetName())
			for _, line := range lines {
				c.UI.Infof(line.Message)
			}
			c.UI.Infof("")
		case *services.ServiceGroupConfig:
			return c.tipLogServicesOrGroups(v.Children(), lineCount)
		}
	}
	return nil
}

func getLastLinesOfLog(logDir string, service *services.ServiceConfig, lineCount int) ([]servicelogs.LogLine, error) {
	runLog := service.GetRunLog(logDir)
	logFile, err := os.Open(runLog)
	defer logFile.Close()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	var (
		lines   []servicelogs.LogLine
		curLine string
		cursor  int64
	)
	stat, _ := logFile.Stat()
	filesize := stat.Size()
	for len(lines) < lineCount {
		cursor--
		_, err := logFile.Seek(cursor, io.SeekEnd)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		char := make([]byte, 1)
		logFile.Read(char)

		if cursor != -1 && (char[0] == 10 || char[0] == 13) {
			lines, err = prependLogLine(curLine, lines)
			if err != nil {
				return nil, errors.WithStack(err)
			}
			curLine = ""
			continue
		}

		curLine = fmt.Sprintf("%s%s", string(char), curLine)

		if cursor == -filesize {
			break
		}
	}

	if curLine != "" {
		lines, err = prependLogLine(curLine, lines)
		if err != nil {
			return nil, errors.WithStack(err)
		}
	}

	return lines, nil
}

func prependLogLine(line string, lines []servicelogs.LogLine) ([]servicelogs.LogLine, error) {
	parsedLine, err := servicelogs.ParseLogLine(line)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if parsedLine.Stream == "stdout" {
		return append([]servicelogs.LogLine{parsedLine}, lines...), nil
	}
	return lines, nil
}
