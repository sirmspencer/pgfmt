package format

import (
	"strings"
	"unicode/utf8"
)

// visibleWidth counts the characters a reader sees, ignoring item marks
func visibleWidth(text string) int {
	return utf8.RuneCountInString(markPattern.ReplaceAllString(text, ""))
}

// alignedItems lays out rows one per line with trailing commas, padding each
// cell except a row's last so that cells line up in columns. a cell that spans
// several lines cannot be aligned, so its row is printed as written.
// learn: rows is a slice of slices: rows[i] is one line's cells, rows[i][j] one column of it.
func alignedItems(rows [][]string) string {
	lines := make([][]string, len(rows))
	for index, row := range rows {
		var cells []string
		for _, cell := range row {
			if cell != "" {
				cells = append(cells, cell)
			}
		}
		if strings.Contains(strings.Join(cells, " "), "\n") {
			cells = []string{strings.Join(cells, " ")}
		}
		lines[index] = cells
	}
	// a column is as wide as its widest cell among rows that have a cell after it
	widths := map[int]int{}
	for _, cells := range lines {
		for column := 0; column < len(cells)-1; column++ {
			widths[column] = max(widths[column], visibleWidth(cells[column]))
		}
	}
	out := make([]string, len(lines))
	for index, cells := range lines {
		var line strings.Builder
		for column, cell := range cells {
			line.WriteString(cell)
			if column < len(cells)-1 {
				line.WriteString(strings.Repeat(" ", widths[column]-visibleWidth(cell)+1))
			}
		}
		out[index] = line.String()
		if index < len(lines)-1 {
			out[index] += ","
		}
	}
	return strings.Join(out, "\n")
}

// alignedClause is clause for items split into cells: one item stays on the
// keyword's line, several are aligned one per line
func (p *printer) alignedClause(keyword string, rows [][]string) string {
	if len(rows) == 1 {
		return keyword + " " + joinCells(rows[0])
	}
	return p.keywordMark(rows[0][0]) + keyword + "\n" + indent(alignedItems(rows))
}

// joinCells prints a row's cells separated by single spaces
func joinCells(cells []string) string {
	var parts []string
	for _, cell := range cells {
		if cell != "" {
			parts = append(parts, cell)
		}
	}
	return strings.Join(parts, " ")
}
