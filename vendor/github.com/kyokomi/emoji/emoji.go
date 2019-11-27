// Package emoji terminal output.
package emoji

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"regexp"
	"unicode"
)

//go:generate generateEmojiCodeMap -pkg emoji

// Replace Padding character for emoji.
const (
	ReplacePadding = " "
)

// CodeMap gets the underlying map of emoji.
func CodeMap() map[string]string {
	return emojiCodeMap
}

// regular expression that matches :flag-[countrycode]:
var flagRegexp = regexp.MustCompile(":flag-([a-z]{2}):")

func emojize(x string) string {
	str, ok := emojiCodeMap[x]
	if ok {
		return str + ReplacePadding
	}
	if match := flagRegexp.FindStringSubmatch(x); len(match) == 2 {
		return regionalIndicator(match[1][0]) + regionalIndicator(match[1][1])
	}
	return x
}

// regionalIndicator maps a lowercase letter to a unicode regional indicator
func regionalIndicator(i byte) string {
	return string('\U0001F1E6' + rune(i) - 'a')
}

func replaseEmoji(input *bytes.Buffer) string {
	emoji := bytes.NewBufferString(":")
	for {
		i, _, err := input.ReadRune()
		if err != nil {
			// not replase
			return emoji.String()
		}

		if i == ':' && emoji.Len() == 1 {
			return emoji.String() + replaseEmoji(input)
		}

		emoji.WriteRune(i)
		switch {
		case unicode.IsSpace(i):
			return emoji.String()
		case i == ':':
			return emojize(emoji.String())
		}
	}
}

func compile(x string) string {
	if x == "" {
		return ""
	}

	input := bytes.NewBufferString(x)
	output := bytes.NewBufferString("")

	for {
		i, _, err := input.ReadRune()
		if err != nil {
			break
		}
		switch i {
		default:
			output.WriteRune(i)
		case ':':
			output.WriteString(replaseEmoji(input))
		}
	}
	return output.String()
}

func compileValues(a *[]interface{}) {
	for i, x := range *a {
		if str, ok := x.(string); ok {
			(*a)[i] = compile(str)
		}
	}
}

// Print is fmt.Print which supports emoji
func Print(a ...interface{}) (int, error) {
	compileValues(&a)
	return fmt.Print(a...)
}

// Println is fmt.Println which supports emoji
func Println(a ...interface{}) (int, error) {
	compileValues(&a)
	return fmt.Println(a...)
}

// Printf is fmt.Printf which supports emoji
func Printf(format string, a ...interface{}) (int, error) {
	format = compile(format)
	compileValues(&a)
	return fmt.Printf(format, a...)
}

// Fprint is fmt.Fprint which supports emoji
func Fprint(w io.Writer, a ...interface{}) (int, error) {
	compileValues(&a)
	return fmt.Fprint(w, a...)
}

// Fprintln is fmt.Fprintln which supports emoji
func Fprintln(w io.Writer, a ...interface{}) (int, error) {
	compileValues(&a)
	return fmt.Fprintln(w, a...)
}

// Fprintf is fmt.Fprintf which supports emoji
func Fprintf(w io.Writer, format string, a ...interface{}) (int, error) {
	format = compile(format)
	compileValues(&a)
	return fmt.Fprintf(w, format, a...)
}

// Sprint is fmt.Sprint which supports emoji
func Sprint(a ...interface{}) string {
	compileValues(&a)
	return fmt.Sprint(a...)
}

// Sprintf is fmt.Sprintf which supports emoji
func Sprintf(format string, a ...interface{}) string {
	format = compile(format)
	compileValues(&a)
	return fmt.Sprintf(format, a...)
}

// Errorf is fmt.Errorf which supports emoji
func Errorf(format string, a ...interface{}) error {
	compileValues(&a)
	return errors.New(Sprintf(format, a...))
}
