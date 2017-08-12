package output

import (
	"fmt"
	"io"
	"time"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/yext/edward/tracker"
)

type CompletionRenderer struct {
	indent     string
	minSpacing int
	targetTask tracker.Task
}

func NewCompletionRenderer(task tracker.Task) *CompletionRenderer {
	return &CompletionRenderer{
		indent:     "  ",
		minSpacing: 3,
		targetTask: task,
	}
}

func (r *CompletionRenderer) Render(w io.Writer) error {
	task := r.targetTask.Lineage()[0]
	err := errors.WithStack(r.doRenderWithPrefix("", 0, w, task))
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func extendPrefix(prefix string, child tracker.Task) string {
	if prefix == "" {
		return child.Name()
	}
	return fmt.Sprintf("%v > %v", prefix, child.Name())
}

func (r *CompletionRenderer) doRenderWithPrefix(prefix string, maxNameWidth int, w io.Writer, task tracker.Task) error {
	newMax := r.getLongestName(w, prefix, task)
	if newMax > maxNameWidth {
		maxNameWidth = newMax
	}

	children := task.Children()
	for _, child := range children {
		r.doRenderWithPrefix(extendPrefix(prefix, child), maxNameWidth, w, child)
	}

	if task != r.targetTask {
		return nil
	}

	ts := task.State()
	// Print name
	nameFormat := fmt.Sprintf("%%-%ds", maxNameWidth+r.minSpacing)
	fmt.Fprintf(w, nameFormat, fmt.Sprintf("%v:", prefix))

	tmpOutput := color.Output
	defer func() {
		color.Output = tmpOutput
	}()
	color.Output = w
	fmt.Fprint(w, "[")
	switch ts {
	case tracker.TaskStateSuccess:
		color.Set(color.FgGreen)
		fmt.Fprint(w, "OK")
	case tracker.TaskStateFailed:
		color.Set(color.FgRed)
		fmt.Fprint(w, "Failed")
	case tracker.TaskStateWarning:
		color.Set(color.FgYellow)
		fmt.Fprint(w, "Warning")
	case tracker.TaskStatePending:
		color.Set(color.FgCyan)
		fmt.Fprint(w, "Pending")
	default:
		color.Set(color.FgCyan)
		fmt.Fprint(w, "In Progress")
	}
	color.Unset()
	fmt.Fprint(w, "]")
	if ts != tracker.TaskStateInProgress && ts != tracker.TaskStatePending {
		fmt.Fprintf(w, " (%v)", autoRoundTime(task.Duration()))
	}
	fmt.Fprintln(w)
	if ts == tracker.TaskStateFailed || ts == tracker.TaskStateWarning {
		for _, line := range task.Messages() {
			fmt.Fprintln(w, line)
		}
	}

	return nil
}

func (r *CompletionRenderer) getLongestName(w io.Writer, prefix string, task tracker.Task) int {
	children := task.Children()
	var max = len(prefix)
	for _, child := range children {
		childMax := r.getLongestName(w, extendPrefix(prefix, child), child)
		if childMax > max {
			max = childMax
		}
	}
	return max
}

func autoRoundTime(d time.Duration) time.Duration {
	if d > time.Hour {
		return roundTime(d, time.Second)
	}
	if d > time.Minute {
		return roundTime(d, time.Second)
	}
	if d > time.Second {
		return roundTime(d, time.Millisecond)
	}
	if d > time.Millisecond {
		return roundTime(d, time.Microsecond)
	}
	return d
}

// Based on the example at https://play.golang.org/p/QHocTHl8iR
func roundTime(d, r time.Duration) time.Duration {
	if r <= 0 {
		return d
	}
	neg := d < 0
	if neg {
		d = -d
	}
	if m := d % r; m+m < r {
		d = d - m
	} else {
		d = d + r - m
	}
	if neg {
		return -d
	}
	return d
}
