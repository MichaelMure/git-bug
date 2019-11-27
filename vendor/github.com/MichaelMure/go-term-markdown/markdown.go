package markdown

import (
	"fmt"
	"io"
	"os"
	"strings"

	md "github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/parser"
)

func Render(source string, lineWidth int, leftPad int, opts ...Options) []byte {
	extensions := parser.CommonExtensions
	extensions |= parser.LaxHTMLBlocks          // more in HTMLBlock, less in HTMLSpan
	extensions |= parser.NoEmptyLineBeforeBlock // no need for NL before a list

	p := parser.NewWithExtensions(extensions)
	nodes := md.Parse([]byte(source), p)
	renderer := newRenderer(lineWidth, leftPad, opts...)

	// astRenderer, err := newAstRenderer()
	// if err != nil {
	// 	panic(err)
	// }
	// md.Render(nodes, astRenderer)

	return md.Render(nodes, renderer)
}

var _ md.Renderer = &astRenderer{}

type astRenderer struct {
	set map[string]struct{}
	f   *os.File
}

func newAstRenderer() (*astRenderer, error) {
	f, err := os.OpenFile("/tmp/ast.puml", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return nil, err
	}

	_, _ = fmt.Fprintln(f, "(*) --> Document")

	return &astRenderer{
		f:   f,
		set: make(map[string]struct{}),
	}, nil
}

func (a *astRenderer) RenderNode(w io.Writer, node ast.Node, entering bool) ast.WalkStatus {
	if entering {
		for _, child := range node.GetChildren() {
			str := fmt.Sprintf("%T --> %T\n", node, child)
			if _, has := a.set[str]; !has {
				a.set[str] = struct{}{}
				_, _ = fmt.Fprintf(a.f, strings.Replace(str, "*ast.", "", -1))
			}
		}
	}

	return ast.GoToNext
}

func (a *astRenderer) RenderHeader(w io.Writer, ast ast.Node) {
	// _, _ = fmt.Fprintln(a.f, "@startuml")
}

func (a *astRenderer) RenderFooter(w io.Writer, ast ast.Node) {
	// _, _ = fmt.Fprintln(a.f, "@enduml")
	_ = a.f.Close()
}
