package terminal

import (
	"os"
	"strings"
)

type tablePrinter interface {
	SetHeader(header []string)
	Append(columns []string)
	Render()
}

type plainPrinter struct {
	out     *os.File
	headers []string
	rows    [][]string
}

func NewPlainPrinter(out *os.File) *plainPrinter {
	p := new(plainPrinter)
	p.out = out
	return p
}

func (p *plainPrinter) SetHeader(header []string) {
	p.headers = header
}

func (p *plainPrinter) Append(columns []string) {
	p.rows = append(p.rows, columns)
}

func (p *plainPrinter) Render() {
	for _, col := range p.headers {
		p.out.WriteString(strings.ToUpper(col))
		p.out.WriteString("\t")
	}
	p.out.WriteString("\n")
	for _, row := range p.rows {
		for _, col := range row {
			p.out.WriteString(col)
			p.out.WriteString("\t")
		}
		p.out.WriteString("\n")
	}
}
