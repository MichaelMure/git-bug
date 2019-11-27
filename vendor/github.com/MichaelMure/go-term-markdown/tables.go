package markdown

import (
	"io"
	"strings"

	"github.com/MichaelMure/go-term-text"
	"github.com/gomarkdown/markdown/ast"
)

const minColumnCompactedWidth = 5

type tableCell struct {
	content   string
	alignment ast.CellAlignFlags
}

type tableRenderer struct {
	header []tableCell
	body   [][]tableCell
}

func newTableRenderer() *tableRenderer {
	return &tableRenderer{}
}

func (tr *tableRenderer) AddHeaderCell(content string, alignment ast.CellAlignFlags) {
	tr.header = append(tr.header, tableCell{
		content:   content,
		alignment: alignment,
	})
}

func (tr *tableRenderer) NextBodyRow() {
	tr.body = append(tr.body, nil)
}

func (tr *tableRenderer) AddBodyCell(content string) {
	row := tr.body[len(tr.body)-1]
	column := len(row)
	row = append(row, tableCell{
		content:   content,
		alignment: tr.header[column].alignment,
	})
	tr.body[len(tr.body)-1] = row
}

func (tr *tableRenderer) Render(w io.Writer, leftPad int, lineWidth int) {
	columnWidths, truncated := tr.columnWidths(lineWidth - leftPad)
	pad := strings.Repeat(" ", leftPad)

	drawTopLine(w, pad, columnWidths, truncated)

	drawRow(w, pad, tr.header, columnWidths, truncated)

	drawHeaderUnderline(w, pad, columnWidths, truncated)

	for i, row := range tr.body {
		drawRow(w, pad, row, columnWidths, truncated)
		if i != len(tr.body)-1 {
			drawRowLine(w, pad, columnWidths, truncated)
		}
	}

	drawBottomLine(w, pad, columnWidths, truncated)
}

func (tr *tableRenderer) columnWidths(lineWidth int) (widths []int, truncated bool) {
	maxWidth := make([]int, len(tr.header))

	for i, cell := range tr.header {
		maxWidth[i] = max(maxWidth[i], text.MaxLineLen(cell.content))
	}

	for _, row := range tr.body {
		for i, cell := range row {
			maxWidth[i] = max(maxWidth[i], text.MaxLineLen(cell.content))
		}
	}

	sumWidth := 1
	minWidth := 1
	for _, width := range maxWidth {
		sumWidth += width + 1
		minWidth += min(width, minColumnCompactedWidth) + 1
	}

	// Strategy 1: the easy case, content is not large enough to overflow
	if sumWidth <= lineWidth {
		return maxWidth, false
	}

	// Strategy 2: overflow, but still enough room
	if minWidth < lineWidth {
		return tr.overflowColumnWidths(lineWidth, maxWidth), false
	}

	// Strategy 3: too much columns, we need to truncate
	return tr.truncateColumnWidths(lineWidth, maxWidth), true
}

func (tr *tableRenderer) overflowColumnWidths(lineWidth int, maxWidth []int) []int {
	// We have an overflow. First, we take as is the columns that are thinner
	// than the space equally divided.
	// Integer division, rounded lower.
	available := lineWidth - len(tr.header) - 1
	fairSpace := available / len(tr.header)

	result := make([]int, len(tr.header))
	remainingColumn := len(tr.header)

	for i, width := range maxWidth {
		if width <= fairSpace {
			result[i] = width
			available -= width
			remainingColumn--
		} else {
			// Mark the column as non-allocated yet
			result[i] = -1
		}
	}

	// Now we allocate evenly the remaining space to the remaining columns
	for i, width := range result {
		if width == -1 {
			width = available / remainingColumn
			result[i] = width
			available -= width
			remainingColumn--
		}
	}

	return result
}

func (tr *tableRenderer) truncateColumnWidths(lineWidth int, maxWidth []int) []int {
	var result []int
	used := 1

	// Pack as much column as possible without compacting them too much
	for _, width := range maxWidth {
		w := min(width, minColumnCompactedWidth)

		if used+w+1 > lineWidth {
			return result
		}

		result = append(result, w)
		used += w + 1
	}

	return result
}

func drawTopLine(w io.Writer, pad string, columnWidths []int, truncated bool) {
	_, _ = w.Write([]byte(pad))
	_, _ = w.Write([]byte("┌"))
	for i, width := range columnWidths {
		_, _ = w.Write([]byte(strings.Repeat("─", width)))
		if i != len(columnWidths)-1 {
			_, _ = w.Write([]byte("┬"))
		}
	}
	_, _ = w.Write([]byte("┐"))
	if truncated {
		_, _ = w.Write([]byte("…"))
	}
	_, _ = w.Write([]byte("\n"))
}

func drawHeaderUnderline(w io.Writer, pad string, columnWidths []int, truncated bool) {
	_, _ = w.Write([]byte(pad))
	_, _ = w.Write([]byte("╞"))
	for i, width := range columnWidths {
		_, _ = w.Write([]byte(strings.Repeat("═", width)))
		if i != len(columnWidths)-1 {
			_, _ = w.Write([]byte("╪"))
		}
	}
	_, _ = w.Write([]byte("╡"))
	if truncated {
		_, _ = w.Write([]byte("…"))
	}
	_, _ = w.Write([]byte("\n"))
}

func drawBottomLine(w io.Writer, pad string, columnWidths []int, truncated bool) {
	_, _ = w.Write([]byte(pad))
	_, _ = w.Write([]byte("└"))
	for i, width := range columnWidths {
		_, _ = w.Write([]byte(strings.Repeat("─", width)))
		if i != len(columnWidths)-1 {
			_, _ = w.Write([]byte("┴"))
		}
	}
	_, _ = w.Write([]byte("┘"))
	if truncated {
		_, _ = w.Write([]byte("…"))
	}
	_, _ = w.Write([]byte("\n"))
}

func drawRowLine(w io.Writer, pad string, columnWidths []int, truncated bool) {
	_, _ = w.Write([]byte(pad))
	_, _ = w.Write([]byte("├"))
	for i, width := range columnWidths {
		_, _ = w.Write([]byte(strings.Repeat("─", width)))
		if i != len(columnWidths)-1 {
			_, _ = w.Write([]byte("┼"))
		}
	}
	_, _ = w.Write([]byte("┤"))
	if truncated {
		_, _ = w.Write([]byte("…"))
	}
	_, _ = w.Write([]byte("\n"))
}

func drawRow(w io.Writer, pad string, cells []tableCell, columnWidths []int, truncated bool) {
	contents := make([][]string, len(cells))

	// As we draw the row line by line, we need a way to reset and recover
	// the formatting when we alternate between cells. To do that, we accumulate
	// the ongoing series of ANSI escape sequence for each cell and output them
	// again each time we switch to the next cell so we end up in the exact same
	// state. Inefficient but works.
	formatting := make([]strings.Builder, len(cells))

	accFormatting := func(cellIndex int, items []text.EscapeItem) {
		for _, item := range items {
			formatting[cellIndex].WriteString(item.Item)
		}
	}

	maxHeight := 0

	// Wrap each cell content into multiple lines, depending on
	// how wide each cell is.
	for i, cell := range cells {
		wrapped, lines := text.Wrap(cell.content, columnWidths[i])
		contents[i] = strings.Split(wrapped, "\n")
		maxHeight = max(maxHeight, lines)
	}

	// Draw the row line by line
	for i := 0; i < maxHeight; i++ {
		_, _ = w.Write([]byte(pad))
		_, _ = w.Write([]byte("│"))
		for j, width := range columnWidths {
			content := ""
			if len(contents[j]) > i {
				content = contents[j][i]
				trimmed := text.TrimSpace(content)

				switch cells[j].alignment {
				case ast.TableAlignmentLeft, 0:
					_, _ = w.Write([]byte(formatting[j].String()))
					_, _ = w.Write([]byte(trimmed))
					_, _ = w.Write([]byte(resetAll))
					_, _ = w.Write([]byte(strings.Repeat(" ", width-text.Len(trimmed))))

				case ast.TableAlignmentCenter:
					spaces := width - text.Len(trimmed)
					_, _ = w.Write([]byte(strings.Repeat(" ", spaces/2)))
					_, _ = w.Write([]byte(formatting[j].String()))
					_, _ = w.Write([]byte(trimmed))
					_, _ = w.Write([]byte(resetAll))
					_, _ = w.Write([]byte(strings.Repeat(" ", spaces-(spaces/2))))

				case ast.TableAlignmentRight:
					_, _ = w.Write([]byte(strings.Repeat(" ", width-text.Len(trimmed))))
					_, _ = w.Write([]byte(formatting[j].String()))
					_, _ = w.Write([]byte(trimmed))
					_, _ = w.Write([]byte(resetAll))
				}

				// extract and accumulate the formatting
				_, seqs := text.ExtractTermEscapes(content)
				accFormatting(j, seqs)
			} else {
				padding := strings.Repeat(" ", width-text.Len(content))
				_, _ = w.Write([]byte(padding))
			}
			_, _ = w.Write([]byte("│"))
		}
		if truncated {
			_, _ = w.Write([]byte("…"))
		}
		_, _ = w.Write([]byte("\n"))
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
