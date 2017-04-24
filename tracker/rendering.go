package tracker

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/pkg/errors"
)

type Renderer interface {
	Render(io.Writer, Task) error
}

type ANSIRenderer struct {
	indent     string
	minSpacing int
}

func NewAnsiRenderer() *ANSIRenderer {
	return &ANSIRenderer{
		indent:     "  ",
		minSpacing: 3,
	}
}

func (r *ANSIRenderer) Render(w io.Writer, task Task) error {
	return errors.WithStack(r.renderWithIndent(0, 0, w, task))
}

func (r *ANSIRenderer) getDisplayName(task Task) string {
	children := task.Children()
	if len(children) == 1 {
		child := children[0]
		return fmt.Sprintf("%v > %v", task.Name(), r.getDisplayName(child))
	}
	return fmt.Sprintf("%v", task.Name())
}

func (r *ANSIRenderer) renderWithIndent(i int, maxNameWidth int, w io.Writer, task Task) error {
	newMax := r.getLongestNameWithIndent(w, i, task)
	if newMax > maxNameWidth {
		maxNameWidth = newMax
	}

	name := r.getDisplayName(task)

	children := task.Children()
	if len(task.Name()) == 0 {
		for _, child := range children {
			r.renderWithIndent(i, maxNameWidth, w, child)
		}
		return nil
	}
	ts := task.State()

	// Print indent
	fmt.Fprint(w, strings.Repeat(r.indent, i))

	// Print name
	nameFormat := fmt.Sprintf("%%-%ds", maxNameWidth+r.minSpacing-i*len(r.indent))
	fmt.Fprintf(w, nameFormat, fmt.Sprintf("%v:", name))

	if len(children) > 1 {
		fmt.Fprintln(w)
		for _, child := range children {
			r.renderWithIndent(i+1, maxNameWidth, w, child)
		}
		return nil
	}

	tmpOutput := color.Output
	defer func() {
		color.Output = tmpOutput
	}()
	color.Output = w
	fmt.Fprint(w, "[")
	switch ts {
	case TaskStateSuccess:
		color.Set(color.FgGreen)
		fmt.Fprint(w, "OK")
	case TaskStateFailed:
		color.Set(color.FgRed)
		fmt.Fprint(w, "Failed")
	case TaskStateWarning:
		color.Set(color.FgYellow)
		fmt.Fprint(w, "Warning")
	case TaskStatePending:
		color.Set(color.FgCyan)
		fmt.Fprint(w, "Pending")
	default:
		color.Set(color.FgCyan)
		fmt.Fprint(w, "In Progress")
	}
	color.Unset()
	fmt.Fprint(w, "]")
	if ts != TaskStateInProgress && ts != TaskStatePending {
		fmt.Fprintf(w, " (%v)", autoRoundTime(task.Duration()))
	}
	fmt.Fprintln(w)
	if ts == TaskStateFailed || ts == TaskStateWarning {
		for _, line := range task.Messages() {
			fmt.Fprint(w, strings.Repeat(r.indent, i))
			fmt.Fprintln(w, line)
		}
	}

	return nil
}

func (r *ANSIRenderer) getLongestNameWithIndent(w io.Writer, i int, task Task) int {
	children := task.Children()
	name := r.getDisplayName(task)
	var max = len(name) + i*len(r.indent)
	for _, child := range children {
		childMax := r.getLongestNameWithIndent(w, i+1, child)
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
