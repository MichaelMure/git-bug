package markdown

var headingShades = []func(a ...interface{}) string{
	GreenBold,
	GreenBold,
	HiGreen,
	Green,
}

// Return the color function corresponding to the level.
// Beware, level start counting from 1.
func headingShade(level int) func(a ...interface{}) string {
	if level < 1 {
		level = 1
	}
	if level > len(headingShades) {
		level = len(headingShades)
	}
	return headingShades[level-1]
}

var quoteShades = []func(a ...interface{}) string{
	GreenBold,
	GreenBold,
	HiGreen,
	Green,
}

// Return the color function corresponding to the level.
func quoteShade(level int) func(a ...interface{}) string {
	if level < 1 {
		level = 1
	}
	if level > len(quoteShades) {
		level = len(quoteShades)
	}
	return quoteShades[level-1]
}
