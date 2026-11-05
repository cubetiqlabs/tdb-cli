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
	headerWidths := make([]int, columnCount)
	for i := 0; i < columnCount; i++ {
		w := runewidth.StringWidth(headers[i])
		widths[i] = w
		headerWidths[i] = w
	}

	rowWidths := make([][]int, len(rows))
	for i, row := range rows {
		rw := make([]int, columnCount)
		rowWidths[i] = rw
		for j := 0; j < columnCount; j++ {
			cell := ""
			if j < len(row) {
				cell = row[j]
			}
			w := runewidth.StringWidth(cell)
			rw[j] = w
			if w > widths[j] {
				widths[j] = w
			}
		}
	}

	top := buildBorder(widths, "┌", "┬", "┐")
	fmt.Fprintln(out, styles.separator(top))

	headerLine := buildRowLine(headers, widths, headerWidths)
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
		line := buildRowLine(cells, widths, rowWidths[idx])
		fmt.Fprintln(out, styles.row(line, idx%2 == 1))
	}

	bottom := buildBorder(widths, "└", "┴", "┘")
	fmt.Fprintln(out, styles.separator(bottom))
}

func buildRowLine(cells []string, widths []int, textWidths []int) string {
	// Pre-calculate exact capacity
	var totalLen int = 3 // Initial "│"
	for _, w := range widths {
		totalLen += 1 + w + 4 // " " + padded_cell + " │"
	}

	var b strings.Builder
	b.Grow(totalLen)
	b.WriteString("│")
	for i, w := range widths {
		cell := ""
		tw := 0
		if i < len(cells) {
			cell = cells[i]
			if i < len(textWidths) {
				tw = textWidths[i]
			} else {
				tw = runewidth.StringWidth(cell)
			}
		}

		b.WriteString(" ")
		b.WriteString(cell)

		// Optimization: Inline padding and avoid strings.Repeat allocations
		pad := w - tw
		if pad > 0 {
			for j := 0; j < pad; j++ {
				b.WriteByte(' ')
			}
		}
		b.WriteString(" │")
	}
	return b.String()
}

func buildBorder(widths []int, left, mid, right string) string {
	var totalLen int
	totalLen += len(left) + len(right)
	if len(widths) > 0 {
		totalLen += (len(widths) - 1) * len(mid)
		for _, w := range widths {
			totalLen += (w + 2) * 3 // '─' is 3 bytes in UTF-8
		}
	}

	var b strings.Builder
	b.Grow(totalLen)
	b.WriteString(left)
	for i, w := range widths {
		if i > 0 {
			b.WriteString(mid)
		}
		// Optimization: avoid strings.Repeat allocations
		for j := 0; j < w+2; j++ {
			b.WriteString("─")
		}
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
