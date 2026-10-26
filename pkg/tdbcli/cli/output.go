package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/mattn/go-runewidth"
	"github.com/spf13/cobra"
)

const (
	ansiReset     = "\033[0m"
	ansiHeader    = "\033[1;36m"
	ansiSeparator = "\033[2m"
	ansiRowAlt    = "\033[90m"
)

type tableStyles struct {
	enabled bool
}

func newTableStyles(w io.Writer) tableStyles {
	return tableStyles{enabled: supportsANSI(w)}
}

func supportsANSI(w io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	if os.Getenv("FORCE_COLOR") != "" {
		return true
	}
	type fdWriter interface {
		Fd() uintptr
	}
	if f, ok := w.(fdWriter); ok {
		fd := f.Fd()
		return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
	}
	return false
}

func (s tableStyles) header(text string) string {
	if !s.enabled {
		return text
	}
	return ansiHeader + text + ansiReset
}

func (s tableStyles) separator(text string) string {
	if !s.enabled {
		return text
	}
	return ansiSeparator + text + ansiReset
}

func (s tableStyles) row(text string, odd bool) string {
	if !s.enabled {
		return text
	}
	if odd {
		return ansiRowAlt + text + ansiReset
	}
	return text
}

func renderTable(cmd *cobra.Command, headers []string, rows [][]string) {
	out := cmd.OutOrStdout()
	styles := newTableStyles(out)

	columnCount := len(headers)
	for _, row := range rows {
		if len(row) > columnCount {
			columnCount = len(row)
		}
	}
	if len(headers) < columnCount {
		extra := make([]string, columnCount-len(headers))
		headers = append(headers, extra...)
	}

	widths := make([]int, columnCount)
	for i := 0; i < columnCount; i++ {
		widths[i] = runewidth.StringWidth(headers[i])
	}
	for _, row := range rows {
		for j := 0; j < columnCount; j++ {
			cell := ""
			if j < len(row) {
				cell = row[j]
			}
			if w := runewidth.StringWidth(cell); w > widths[j] {
				widths[j] = w
			}
		}
	}

	top := buildBorder(widths, "┌", "┬", "┐")
	fmt.Fprintln(out, styles.separator(top))

	headerLine := buildRowLine(headers, widths)
	fmt.Fprintln(out, styles.header(headerLine))

	if len(rows) > 0 {
		mid := buildBorder(widths, "├", "┼", "┤")
		fmt.Fprintln(out, styles.separator(mid))
	}

	for idx, row := range rows {
		cells := make([]string, columnCount)
		for j := 0; j < columnCount; j++ {
			if j < len(row) {
				cells[j] = row[j]
			} else {
				cells[j] = ""
			}
		}
		line := buildRowLine(cells, widths)
		fmt.Fprintln(out, styles.row(line, idx%2 == 1))
	}

	bottom := buildBorder(widths, "└", "┴", "┘")
	fmt.Fprintln(out, styles.separator(bottom))
}

func buildRowLine(cells []string, widths []int) string {
	var b strings.Builder

	// Pre-calculate capacity to avoid reallocations
	// "│" + (" " + padded_cell + " │") per column
	totalLen := 3 // len("│") is 3 bytes
	for _, w := range widths {
		// w + 4 bytes for padding, spaces and border char: " " (1) + w + " │" (4)
		totalLen += w + 5
	}
	b.Grow(totalLen)

	b.WriteString("│")
	for i, w := range widths {
		cell := ""
		if i < len(cells) {
			cell = cells[i]
		}
		padded := padCell(cell, w)
		b.WriteString(" ")
		b.WriteString(padded)
		b.WriteString(" │")
	}
	return b.String()
}

// buildBorder constructs a table border utilizing a pre-sized strings.Builder
// to minimize allocations and overhead.
func buildBorder(widths []int, left, mid, right string) string {
	var b strings.Builder
	totalLen := len(left) + len(right)
	for _, w := range widths {
		totalLen += (w + 2) * 3 // '─' is 3 bytes
	}
	if len(widths) > 1 {
		totalLen += (len(widths) - 1) * len(mid)
	}
	b.Grow(totalLen)

	b.WriteString(left)
	for i, w := range widths {
		if i > 0 {
			b.WriteString(mid)
		}
		b.WriteString(strings.Repeat("─", w+2))
	}
	b.WriteString(right)
	return b.String()
}

func padCell(text string, width int) string {
	pad := width - runewidth.StringWidth(text)
	if pad > 0 {
		return text + strings.Repeat(" ", pad)
	}
	return text
}
