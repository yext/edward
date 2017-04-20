package tracker

import (
	"fmt"
	"io"
	"strings"

	"github.com/fatih/color"
	"github.com/pkg/errors"
)

type Renderer interface {
	Render(io.Writer, Task) error
}

type ANSIRenderer struct {
	indent string
}

func NewAnsiRenderer() *ANSIRenderer {
	return &ANSIRenderer{
		indent: "  ",
	}
}

func (r *ANSIRenderer) Render(w io.Writer, task Task) error {
	return errors.WithStack(r.renderWithIndent(0, 0, w, task))
}

func (r *ANSIRenderer) renderWithIndent(i int, maxNameWidth int, w io.Writer, task Task) error {
	newMax := r.getLongestNameWithIndent(w, i, task)
	if newMax > maxNameWidth {
		maxNameWidth = newMax
	}
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
	nameFormat := fmt.Sprintf("%%-%ds", maxNameWidth+7-i*len(r.indent))
	fmt.Fprintf(w, nameFormat, fmt.Sprintf("%v:", task.Name()))

	if len(children) > 0 {
		fmt.Fprintln(w)
		for _, child := range children {
			r.renderWithIndent(i+1, maxNameWidth, w, child)
		}
	} else {
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
		default:
			color.Set(color.FgCyan)
			fmt.Fprint(w, "In Progress")
		}
		color.Unset()
		fmt.Fprint(w, "]")
		fmt.Fprintln(w)
		if ts == TaskStateFailed || ts == TaskStateWarning {
			for _, line := range task.Messages() {
				fmt.Fprint(w, strings.Repeat(r.indent, i))
				fmt.Fprintln(w, line)
			}
		}
	}
	return nil
}

func (r *ANSIRenderer) getLongestNameWithIndent(w io.Writer, i int, task Task) int {
	children := task.Children()
	var max = len(task.Name()) + i*len(r.indent)
	for _, child := range children {
		childMax := r.getLongestNameWithIndent(w, i+1, child)
		if childMax > max {
			max = childMax
		}
	}
	return max
}
