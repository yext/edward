package output

import (
	"fmt"
	"io"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/yext/edward/tracker"
)

type InProgressRenderer struct {
	indent     string
	minSpacing int
}

func NewInProgressRenderer() *InProgressRenderer {
	return &InProgressRenderer{
		indent:     "  ",
		minSpacing: 3,
	}
}

func (r *InProgressRenderer) Render(w io.Writer, task tracker.Task) error {
	return errors.WithStack(r.doRenderWithPrefix("", 0, w, task))
}

func (r *InProgressRenderer) doRenderWithPrefix(prefix string, maxNameWidth int, w io.Writer, task tracker.Task) error {
	newMax := r.getLongestName(w, prefix, task)
	if newMax > maxNameWidth {
		maxNameWidth = newMax
	}

	children := task.Children()
	for _, child := range children {
		r.doRenderWithPrefix(extendPrefix(prefix, child), maxNameWidth, w, child)
	}

	if len(children) != 0 || task.State() != tracker.TaskStateInProgress {
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
	fmt.Fprintln(w)

	return nil
}

func (r *InProgressRenderer) getLongestName(w io.Writer, prefix string, task tracker.Task) int {
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
