/*
Package markdown implements markdown parser and HTML renderer.

It parses markdown into AST format which can be serialized to HTML
(using html.Renderer) or possibly other formats (using alternate renderers).

Convert markdown to HTML

The simplest way to convert markdown document to HTML

  md := []byte("## markdown document")
  html := markdown.ToHTML(md, nil, nil)

Customizing parsing and HTML rendering

You can customize parser and HTML renderer:

	import (
		"github.com/gomarkdown/markdown/parser"
		"github.com/gomarkdown/markdown/renderer"
		"github.com/gomarkdown/markdown"
	)
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs
	p := parser.NewWithExtensions(extensions)

	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)

	md := []byte("markdown text")
	html := markdown.ToHTML(md, p, renderer)

For a cmd-line tool see https://github.com/gomarkdown/mdtohtml
*/
package markdown
