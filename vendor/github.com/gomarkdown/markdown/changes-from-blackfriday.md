## Changes from blackfriday

This library is derived from blackfriday library. Here's a list of changes.

**Redesigned API**

- split into 3 separate packages: ast, parser and html (for html renderer). This makes the API more manageable. It also separates e.g. parser option from renderer options
- changed how AST node is represented from union-like representation (manually keeping track of the type of the node) to using interface{} (which is a Go way to combine an arbitrary value with its type)

**Allow re-using most of html renderer logic**

You can implement your own renderer by implementing `Renderer` interface.

Implementing a full renderer is a lot of work and often you just want to tweak html rendering of few node typs.

I've added a way to hook `Renderer.Render` function in html renderer with a custom function that can take over rendering of specific nodes.

I use it myself to do syntax-highlighting of code snippets.

**Speed up go test**

Running `go test` was really slow (17 secs) because it did a poor man's version of fuzzing by feeding the parser all subsets of test strings in order to find panics
due to incorrect parsing logic.

I've moved that logic to `cmd/crashtest`, so that it can be run on CI but not slow down regular development.

Now `go test` is blazing fast.
