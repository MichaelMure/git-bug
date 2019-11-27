package markdown

import (
	"bytes"
	"fmt"
	stdcolor "image/color"
	"io"
	"math"
	"net/http"
	"os"
	"strings"
	"unicode"

	"github.com/MichaelMure/go-term-text"
	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/formatters"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	"github.com/eliukblau/pixterm/ansimage"
	"github.com/fatih/color"
	md "github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	"github.com/kyokomi/emoji"
	"golang.org/x/net/html"
)

/*

Here are the possible cases for the AST. You can render it using PlantUML.

@startuml

(*) --> Document
BlockQuote --> BlockQuote
BlockQuote --> CodeBlock
BlockQuote --> List
BlockQuote --> Paragraph
Del --> Emph
Del --> Strong
Del --> Text
Document --> BlockQuote
Document --> CodeBlock
Document --> Heading
Document --> HorizontalRule
Document --> HTMLBlock
Document --> List
Document --> Paragraph
Document --> Table
Emph --> Text
Heading --> Code
Heading --> Del
Heading --> Emph
Heading --> HTMLSpan
Heading --> Image
Heading --> Link
Heading --> Strong
Heading --> Text
Image --> Text
Link --> Image
Link --> Text
ListItem --> List
ListItem --> Paragraph
List --> ListItem
Paragraph --> Code
Paragraph --> Del
Paragraph --> Emph
Paragraph --> Hardbreak
Paragraph --> HTMLSpan
Paragraph --> Image
Paragraph --> Link
Paragraph --> Strong
Paragraph --> Text
Strong --> Emph
Strong --> Text
TableBody --> TableRow
TableCell --> Code
TableCell --> Del
TableCell --> Emph
TableCell --> HTMLSpan
TableCell --> Image
TableCell --> Link
TableCell --> Strong
TableCell --> Text
TableHeader --> TableRow
TableRow --> TableCell
Table --> TableBody
Table --> TableHeader

@enduml

*/

var _ md.Renderer = &renderer{}

type renderer struct {
	// maximum line width allowed
	lineWidth int
	// constant left padding to apply
	leftPad int
	// Dithering mode for ansimage
	// Default is fine directly through a terminal
	// DitheringWithBlocks is recommended if a terminal UI library is used
	imageDithering ansimage.DitheringMode

	// all the custom left paddings, without the fixed space from leftPad
	padAccumulator []string

	// one-shot indent for the first line of the inline content
	indent string

	// for Heading, Paragraph, HTMLBlock and TableCell, accumulate the content of
	// the child nodes (Link, Text, Image, formatting ...). The result
	// is then rendered appropriately when exiting the node.
	inlineAccumulator strings.Builder
	inlineAlign       text.Alignment

	// record and render the heading numbering
	headingNumbering headingNumbering

	blockQuoteLevel int

	table *tableRenderer
}

func newRenderer(lineWidth int, leftPad int, opts ...Options) *renderer {
	r := &renderer{
		lineWidth:      lineWidth,
		leftPad:        leftPad,
		padAccumulator: make([]string, 0, 10),
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

func (r *renderer) pad() string {
	return strings.Repeat(" ", r.leftPad) + strings.Join(r.padAccumulator, "")
}

func (r *renderer) addPad(pad string) {
	r.padAccumulator = append(r.padAccumulator, pad)
}

func (r *renderer) popPad() {
	r.padAccumulator = r.padAccumulator[:len(r.padAccumulator)-1]
}

func (r *renderer) RenderNode(w io.Writer, node ast.Node, entering bool) ast.WalkStatus {
	// TODO: remove
	// fmt.Printf("%T, %v\n", node, entering)

	switch node := node.(type) {
	case *ast.Document:
		// Nothing to do

	case *ast.BlockQuote:
		// set and remove a colored bar on the left
		if entering {
			r.blockQuoteLevel++
			r.addPad(quoteShade(r.blockQuoteLevel)("┃ "))
		} else {
			r.blockQuoteLevel--
			r.popPad()
		}

	case *ast.List:
		if next := ast.GetNextNode(node); !entering && next != nil {
			_, parentIsListItem := node.GetParent().(*ast.ListItem)
			_, nextIsList := next.(*ast.List)
			if !nextIsList && !parentIsListItem {
				_, _ = fmt.Fprintln(w)
			}
		}

	case *ast.ListItem:
		// write the prefix, add a padding if needed, and let Paragraph handle the rest
		if entering {
			switch {
			// numbered list
			case node.ListFlags&ast.ListTypeOrdered != 0:
				itemNumber := 1
				siblings := node.GetParent().GetChildren()
				for _, sibling := range siblings {
					if sibling == node {
						break
					}
					itemNumber++
				}
				prefix := fmt.Sprintf("%d. ", itemNumber)
				r.indent = r.pad() + Green(prefix)
				r.addPad(strings.Repeat(" ", text.Len(prefix)))

			// header of a definition
			case node.ListFlags&ast.ListTypeTerm != 0:
				r.inlineAccumulator.WriteString(greenOn)

			// content of a definition
			case node.ListFlags&ast.ListTypeDefinition != 0:
				r.addPad("  ")

			// no flags means it's the normal bullet point list
			default:
				r.indent = r.pad() + Green("• ")
				r.addPad("  ")
			}
		} else {
			switch {
			// numbered list
			case node.ListFlags&ast.ListTypeOrdered != 0:
				r.popPad()

			// header of a definition
			case node.ListFlags&ast.ListTypeTerm != 0:
				r.inlineAccumulator.WriteString(colorOff)

			// content of a definition
			case node.ListFlags&ast.ListTypeDefinition != 0:
				r.popPad()
				_, _ = fmt.Fprintln(w)

			// no flags means it's the normal bullet point list
			default:
				r.popPad()
			}
		}

	case *ast.Paragraph:
		// on exiting, collect and format the accumulated content
		if !entering {
			content := r.inlineAccumulator.String()
			r.inlineAccumulator.Reset()

			var out string
			if r.indent != "" {
				out, _ = text.WrapWithPadIndent(content, r.lineWidth, r.indent, r.pad())
				r.indent = ""
			} else {
				out, _ = text.WrapWithPad(content, r.lineWidth, r.pad())
			}
			_, _ = fmt.Fprint(w, out, "\n")

			// extra line break in some cases
			if next := ast.GetNextNode(node); next != nil {
				switch next.(type) {
				case *ast.Paragraph, *ast.Heading, *ast.HorizontalRule,
					*ast.CodeBlock, *ast.HTMLBlock:
					_, _ = fmt.Fprintln(w)
				}
			}
		}

	case *ast.Heading:
		if !entering {
			r.renderHeading(w, node.Level)
		}

	case *ast.HorizontalRule:
		r.renderHorizontalRule(w)

	case *ast.Emph:
		if entering {
			r.inlineAccumulator.WriteString(italicOn)
		} else {
			r.inlineAccumulator.WriteString(italicOff)
		}

	case *ast.Strong:
		if entering {
			r.inlineAccumulator.WriteString(boldOn)
		} else {
			r.inlineAccumulator.WriteString(boldOff)
		}

	case *ast.Del:
		if entering {
			r.inlineAccumulator.WriteString(crossedOutOn)
		} else {
			r.inlineAccumulator.WriteString(crossedOutOff)
		}

	case *ast.Link:
		if entering {
			r.inlineAccumulator.WriteString("[")
			r.inlineAccumulator.WriteString(string(ast.GetFirstChild(node).AsLeaf().Literal))
			r.inlineAccumulator.WriteString("](")
			r.inlineAccumulator.WriteString(Blue(string(node.Destination)))
			if len(node.Title) > 0 {
				r.inlineAccumulator.WriteString(" ")
				r.inlineAccumulator.WriteString(string(node.Title))
			}
			r.inlineAccumulator.WriteString(")")
			return ast.SkipChildren
		}

	case *ast.Image:
		if entering {
			var title string

			// the alt text/title is weirdly parsed and is actually
			// a child text of this node
			if len(node.Children) == 1 {
				if t, ok := node.Children[0].(*ast.Text); ok {
					title = string(t.Literal)
				}
			}

			info := fmt.Sprintf("![%s](%s)",
				Green(string(node.Destination)), Blue(title))

			switch node.GetParent().(type) {
			case *ast.Paragraph:
				rendered, err := r.renderImage(
					string(node.Destination), title,
					r.lineWidth-r.leftPad,
				)
				if err != nil {
					r.inlineAccumulator.WriteString(Red(fmt.Sprintf("|%s|", err)))
					r.inlineAccumulator.WriteString("\n")
					r.inlineAccumulator.WriteString(info)
					if ast.GetNextNode(node) == nil {
						r.inlineAccumulator.WriteString("\n")
					}
					return ast.SkipChildren
				}

				r.inlineAccumulator.WriteString(rendered)
				r.inlineAccumulator.WriteString(info)
				if ast.GetNextNode(node) == nil {
					r.inlineAccumulator.WriteString("\n")
				}

			default:
				r.inlineAccumulator.WriteString(info)
			}
			return ast.SkipChildren
		}

	case *ast.Text:
		if string(node.Literal) == "\n" {
			break
		}
		content := string(node.Literal)
		if shouldCleanText(node) {
			content = removeLineBreak(content)
		}
		// emoji support !
		emojed := emoji.Sprint(content)
		r.inlineAccumulator.WriteString(emojed)

	case *ast.HTMLBlock:
		r.renderHTMLBlock(w, node)

	case *ast.CodeBlock:
		r.renderCodeBlock(w, node)

	case *ast.Softbreak:
		// not actually implemented in gomarkdown
		r.inlineAccumulator.WriteString("\n")

	case *ast.Hardbreak:
		r.inlineAccumulator.WriteString("\n")

	case *ast.Code:
		r.inlineAccumulator.WriteString(BlueBgItalic(string(node.Literal)))

	case *ast.HTMLSpan:
		r.inlineAccumulator.WriteString(Red(string(node.Literal)))

	case *ast.Table:
		if entering {
			r.table = newTableRenderer()
		} else {
			r.table.Render(w, r.leftPad, r.lineWidth)
			r.table = nil
		}

	case *ast.TableCell:
		if !entering {
			content := r.inlineAccumulator.String()
			r.inlineAccumulator.Reset()

			if node.IsHeader {
				r.table.AddHeaderCell(content, node.Align)
			} else {
				r.table.AddBodyCell(content)
			}
		}

	case *ast.TableHeader:
		// nothing to do

	case *ast.TableBody:
		// nothing to do

	case *ast.TableRow:
		if _, ok := node.Parent.(*ast.TableBody); ok && entering {
			r.table.NextBodyRow()
		}

	default:
		panic(fmt.Sprintf("Unknown node type %T", node))
	}

	return ast.GoToNext
}

func (*renderer) RenderHeader(w io.Writer, node ast.Node) {}

func (*renderer) RenderFooter(w io.Writer, node ast.Node) {}

func (r *renderer) renderHorizontalRule(w io.Writer) {
	_, _ = fmt.Fprintf(w, "%s%s\n\n", r.pad(), strings.Repeat("─", r.lineWidth-r.leftPad))
}

func (r *renderer) renderHeading(w io.Writer, level int) {
	content := r.inlineAccumulator.String()
	r.inlineAccumulator.Reset()

	// render the full line with the headingNumbering
	r.headingNumbering.Observe(level)
	content = fmt.Sprintf("%s %s", r.headingNumbering.Render(), content)
	content = headingShade(level)(content)

	// wrap if needed
	wrapped, _ := text.WrapWithPad(content, r.lineWidth, r.pad())
	_, _ = fmt.Fprintln(w, wrapped)

	// render the underline, if any
	if level == 1 {
		_, _ = fmt.Fprintf(w, "%s%s\n", r.pad(), strings.Repeat("─", r.lineWidth-r.leftPad))
	}

	_, _ = fmt.Fprintln(w)
}

func (r *renderer) renderCodeBlock(w io.Writer, node *ast.CodeBlock) {
	code := string(node.Literal)
	var lexer chroma.Lexer
	// try to get the lexer from the language tag if any
	if len(node.Info) > 0 {
		lexer = lexers.Get(string(node.Info))
	}
	// fallback on detection
	if lexer == nil {
		lexer = lexers.Analyse(code)
	}
	// all failed :-(
	if lexer == nil {
		lexer = lexers.Fallback
	}
	// simplify the lexer output
	lexer = chroma.Coalesce(lexer)

	var formatter chroma.Formatter
	if color.NoColor {
		formatter = formatters.Fallback
	} else {
		formatter = formatters.TTY8
	}

	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		// Something failed, falling back to no highlight render
		r.renderFormattedCodeBlock(w, code)
		return
	}

	buf := &bytes.Buffer{}

	err = formatter.Format(buf, styles.Pygments, iterator)
	if err != nil {
		// Something failed, falling back to no highlight render
		r.renderFormattedCodeBlock(w, code)
		return
	}

	r.renderFormattedCodeBlock(w, buf.String())
}

func (r *renderer) renderFormattedCodeBlock(w io.Writer, code string) {
	// remove the trailing line break
	code = strings.TrimRight(code, "\n")

	r.addPad(GreenBold("┃ "))
	output, _ := text.WrapWithPad(code, r.lineWidth, r.pad())
	r.popPad()

	_, _ = fmt.Fprint(w, output)

	_, _ = fmt.Fprintf(w, "\n\n")
}

func (r *renderer) renderHTMLBlock(w io.Writer, node *ast.HTMLBlock) {
	z := html.NewTokenizer(bytes.NewReader(node.Literal))

	var buf bytes.Buffer

	flushInline := func() {
		if r.inlineAccumulator.Len() <= 0 {
			return
		}
		content := r.inlineAccumulator.String()
		r.inlineAccumulator.Reset()
		out, _ := text.WrapWithPad(content, r.lineWidth, r.pad())
		_, _ = fmt.Fprint(&buf, out, "\n\n")
	}

	for {
		switch z.Next() {
		case html.ErrorToken:
			if z.Err() == io.EOF {
				// normal end of the block
				flushInline()
				_, _ = fmt.Fprint(w, buf.String())
				return
			}
			// if there is another error, fallback to a simple render
			r.inlineAccumulator.Reset()

			content := Red(string(node.Literal))
			out, _ := text.WrapWithPad(content, r.lineWidth, r.pad())
			_, _ = fmt.Fprint(w, out, "\n\n")
			return

		case html.TextToken:
			t := z.Text()
			if strings.TrimSpace(string(t)) == "" {
				continue
			}
			r.inlineAccumulator.Write(t)

		case html.StartTagToken: // <tag ...>
			name, _ := z.TagName()
			switch string(name) {

			case "hr":
				flushInline()
				r.renderHorizontalRule(&buf)

			case "div":
				flushInline()
				// align left by default
				r.inlineAlign = text.AlignLeft
				r.handleDivHTMLAttr(z)

			case "h1", "h2", "h3", "h4", "h5", "h6":
				// handled in closing tag
				flushInline()

			case "img":
				flushInline()
				src, title := getImgHTMLAttr(z)
				rendered, err := r.renderImage(src, title, r.lineWidth-r.leftPad)
				if err != nil {
					r.inlineAccumulator.WriteString(Red(string(z.Raw())))
					continue
				}
				padded := text.LeftPadLines(rendered, r.leftPad)
				_, _ = fmt.Fprintln(&buf, padded)

			// ol + li
			// dl + (dt+dd)
			// ul + li

			// a
			// p

			// details
			// summary

			default:
				r.inlineAccumulator.WriteString(Red(string(z.Raw())))
			}
		case html.EndTagToken: // </tag>
			name, _ := z.TagName()
			switch string(name) {

			case "h1":
				r.renderHeading(&buf, 1)
			case "h2":
				r.renderHeading(&buf, 2)
			case "h3":
				r.renderHeading(&buf, 3)
			case "h4":
				r.renderHeading(&buf, 4)
			case "h5":
				r.renderHeading(&buf, 5)
			case "h6":
				r.renderHeading(&buf, 6)

			case "div":
				content := r.inlineAccumulator.String()
				r.inlineAccumulator.Reset()
				if len(content) == 0 {
					continue
				}
				// remove all line breaks, those are fully managed in HTML
				content = strings.Replace(content, "\n", "", -1)
				content, _ = text.WrapWithPadAlign(content, r.lineWidth, r.pad(), r.inlineAlign)
				_, _ = fmt.Fprint(&buf, content, "\n\n")
				r.inlineAlign = text.NoAlign

			case "hr", "img":
				// handled in opening tag

			default:
				r.inlineAccumulator.WriteString(Red(string(z.Raw())))
			}

		case html.SelfClosingTagToken: // <tag ... />
			name, _ := z.TagName()
			switch string(name) {
			case "hr":
				flushInline()
				r.renderHorizontalRule(&buf)

			default:
				r.inlineAccumulator.WriteString(Red(string(z.Raw())))
			}

		case html.CommentToken, html.DoctypeToken:
			// Not rendered

		default:
			panic("unhandled case")
		}
	}
}

func (r *renderer) handleDivHTMLAttr(z *html.Tokenizer) {
	for {
		key, value, more := z.TagAttr()
		switch string(key) {
		case "align":
			switch string(value) {
			case "left":
				r.inlineAlign = text.AlignLeft
			case "center":
				r.inlineAlign = text.AlignCenter
			case "right":
				r.inlineAlign = text.AlignRight
			}
		}

		if !more {
			break
		}
	}
}

func getImgHTMLAttr(z *html.Tokenizer) (src, title string) {
	for {
		key, value, more := z.TagAttr()
		switch string(key) {
		case "src":
			src = string(value)
		case "alt":
			title = string(value)
		}

		if !more {
			break
		}
	}
	return
}

func (r *renderer) renderImage(dest string, title string, lineWidth int) (string, error) {
	reader, err := imageFromDestination(dest)
	if err != nil {
		return "", fmt.Errorf("failed to open: %v", err)
	}

	x := r.lineWidth - r.leftPad

	if r.imageDithering == ansimage.DitheringWithChars || r.imageDithering == ansimage.DitheringWithBlocks {
		// not sure why this is needed by ansimage
		x *= 4
	}

	img, err := ansimage.NewScaledFromReader(reader, math.MaxInt32, x,
		stdcolor.Black, ansimage.ScaleModeFit, r.imageDithering)

	if err != nil {
		return "", fmt.Errorf("failed to open: %v", err)
	}

	return img.Render(), nil
}

func imageFromDestination(dest string) (io.ReadCloser, error) {
	if strings.HasPrefix(dest, "http://") || strings.HasPrefix(dest, "https://") {
		res, err := http.Get(dest)
		if err != nil {
			return nil, err
		}
		if res.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("http: %v", http.StatusText(res.StatusCode))
		}

		return res.Body, nil
	}

	return os.Open(dest)
}

func removeLineBreak(text string) string {
	lines := strings.Split(text, "\n")

	if len(lines) <= 1 {
		return text
	}

	for i, l := range lines {
		switch i {
		case 0:
			lines[i] = strings.TrimRightFunc(l, unicode.IsSpace)
		case len(lines) - 1:
			lines[i] = strings.TrimLeftFunc(l, unicode.IsSpace)
		default:
			lines[i] = strings.TrimFunc(l, unicode.IsSpace)
		}
	}
	return strings.Join(lines, " ")
}

func shouldCleanText(node ast.Node) bool {
	for node != nil {
		switch node.(type) {
		case *ast.BlockQuote:
			return false

		case *ast.Heading, *ast.Image, *ast.Link,
			*ast.TableCell, *ast.Document, *ast.ListItem:
			return true
		}

		node = node.GetParent()
	}

	panic("bad markdown document or missing case")
}
