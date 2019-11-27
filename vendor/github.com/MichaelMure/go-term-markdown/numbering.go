package markdown

import "strconv"

type headingNumbering struct {
	levels [6]int
}

// Observe register the event of a new level with the given depth and
// adjust the numbering accordingly
func (hn *headingNumbering) Observe(level int) {
	if level <= 0 {
		panic("level start at 1, ask blackfriday why")
	}
	if level > 6 {
		panic("Markdown is limited to 6 levels of heading")
	}

	hn.levels[level-1]++
	for i := level; i < 6; i++ {
		hn.levels[i] = 0
	}
}

// Render render the current headings numbering.
func (hn *headingNumbering) Render() string {
	slice := hn.levels[:]

	// pop the last zero levels
	for i := 5; i >= 0; i-- {
		if hn.levels[i] != 0 {
			break
		}
		slice = slice[:len(slice)-1]
	}

	var result string

	for i := range slice {
		if i > 0 {
			result += "."
		}
		result += strconv.Itoa(slice[i])
	}

	return result
}
