package tracker

import (
	"fmt"
	"io"
	"strings"

	"github.com/pkg/errors"
)

type Renderer interface {
	Render(io.Writer, Task) error
}

type ANSIRenderer struct {
}

func (r *ANSIRenderer) Render(w io.Writer, task Task) error {
	return errors.WithStack(r.renderWithIndent(0, 0, w, task))
}

func (r *ANSIRenderer) renderWithIndent(i int, maxNameWidth int, w io.Writer, task Task) error {
	newMax := r.getLongestNameWithIndent(i, task)
	if newMax > maxNameWidth {
		maxNameWidth = newMax
	}

	var state = "In Progress"
	switch task.State() {
	case TaskStateSuccess:
		state = "OK"
	case TaskStateFailed:
		state = "Failed"
	case TaskStateWarning:
		state = "Warning"
	}
	indents := strings.Repeat("  ", i)

	format := fmt.Sprintf("%%v%%-%ds[%%v]\n", maxNameWidth+7-len(indents))
	fmt.Fprintf(w, format, indents, fmt.Sprintf("%v:", task.Name()), state)
	for _, child := range task.Children() {
		r.renderWithIndent(i+1, maxNameWidth, w, child)
	}
	return nil
}

func (r *ANSIRenderer) getLongestNameWithIndent(i int, task Task) int {
	var max = len(task.Name()) + i*2
	for _, child := range task.Children() {
		childMax := r.getLongestNameWithIndent(i+1, child)
		if childMax > max {
			max = childMax
		}
	}
	return max
}
