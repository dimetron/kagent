package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var update = flag.Bool("update", false, "rewrite docs/architecture/workflow-patterns.md")

const docPath = "../../../../docs/architecture/workflow-patterns.md"

// TestDocIsCurrent keeps the published table identical to the code that the
// other tests run. Regenerate it with:
//
//	go test ./adk/examples/workflowpatterns -run TestDocIsCurrent -update
func TestDocIsCurrent(t *testing.T) {
	rendered, err := renderDoc()
	require.NoError(t, err)
	if *update {
		require.NoError(t, os.WriteFile(docPath, []byte(rendered), 0o644))
		return
	}
	current, err := os.ReadFile(docPath)
	require.NoError(t, err)
	require.Equal(t, rendered, string(current), "docs/architecture/workflow-patterns.md is stale; run the test with -update")
}

// funcSource returns the source of a top-level function, with its comment.
func funcSource(file, name string) (string, error) {
	src, err := os.ReadFile(file)
	if err != nil {
		return "", err
	}
	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, file, src, parser.ParseComments)
	if err != nil {
		return "", err
	}
	for _, decl := range parsed.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != name {
			continue
		}
		start := fn.Pos()
		if fn.Doc != nil {
			start = fn.Doc.Pos()
		}
		return string(src[fset.Position(start).Offset:fset.Position(fn.End()).Offset]), nil
	}
	return "", fmt.Errorf("function %s not found in %s", name, file)
}

func block(lang, body string) string {
	return "```" + lang + "\n" + strings.TrimRight(body, "\n") + "\n```"
}

func renderDoc() (string, error) {
	var b bytes.Buffer
	b.WriteString(docIntro)

	b.WriteString("\n## Building blocks\n\n")
	b.WriteString("Both columns use the same behavior, so only the wiring differs. The Go column wraps a plain function with `step` or `router`; the YAML column names the same function, which the registry exposes through `textFunc` and `routeFunc`. The behaviors themselves (`clean`, `upper`, `classify`, ...) are one-liners in `helpers.go`.\n\n")
	for _, ref := range []struct{ file, name string }{
		{"helpers.go", "step"}, {"helpers.go", "router"}, {"registry.go", "textFunc"}, {"registry.go", "routeFunc"},
	} {
		src, err := funcSource(ref.file, ref.name)
		if err != nil {
			return "", err
		}
		b.WriteString(block("go", src) + "\n\n")
	}

	b.WriteString("## Patterns\n\n")
	b.WriteString("Every row is built twice and run by the tests; the Go and YAML versions must produce identical results. The last line of each Pattern cell is one verified run.\n\n")
	b.WriteString("<table>\n<thead><tr><th>Pattern</th><th>Go (native ADK)</th><th>YAML (declarative)</th></tr></thead>\n<tbody>\n")
	for _, p := range Catalog() {
		goSrc, err := funcSource("patterns.go", p.GoFuncName())
		if err != nil {
			return "", err
		}
		yamlSrc, err := p.YAML()
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&b, "<tr>\n<td valign=\"top\">\n\n**%s**\n\n%s\n\n%s\n\n%s\n\n</td>\n<td valign=\"top\">\n\n%s\n\n</td>\n<td valign=\"top\">\n\n%s\n\n</td>\n</tr>\n",
			p.Title, p.Summary, block("text", p.Shape), verifiedRun(p), block("go", goSrc), block("yaml", yamlSrc))
	}
	b.WriteString("</tbody>\n</table>\n")

	b.WriteString("\n## Rejected graphs\n\n")
	b.WriteString("These look reasonable and fail. Each failure is asserted for both versions, so the YAML validation does not hide a graph the Go API accepts, or the reverse.\n\n")
	b.WriteString("<table>\n<thead><tr><th>Anti-pattern</th><th>Go (native ADK)</th><th>YAML (declarative)</th></tr></thead>\n<tbody>\n")
	for _, a := range AntiPatterns() {
		goCell := "_The ADK Go API accepts this graph; only the declarative validation refuses it._"
		if a.Go != nil {
			src, err := funcSource("patterns.go", a.GoFuncName())
			if err != nil {
				return "", err
			}
			goCell = block("go", src)
		}
		yamlSrc, err := a.YAML()
		if err != nil {
			return "", err
		}
		when := "when built"
		if a.AtRun {
			when = "when run"
		}
		fmt.Fprintf(&b, "<tr>\n<td valign=\"top\">\n\n**%s**\n\n%s\n\nFails %s with:\n\n%s\n\n</td>\n<td valign=\"top\">\n\n%s\n\n</td>\n<td valign=\"top\">\n\n%s\n\n</td>\n</tr>\n",
			a.Title, a.Why, when, block("text", "…"+a.WantErr+"…"), goCell, block("yaml", yamlSrc))
	}
	b.WriteString("</tbody>\n</table>\n")
	b.WriteString(docOutro)
	return b.String(), nil
}

// verifiedRun shows the first documented case of a pattern.
func verifiedRun(p Pattern) string {
	c := p.Cases[0]
	line := fmt.Sprintf("Run: `%q`", c.Input)
	for _, reply := range c.Replies {
		line += fmt.Sprintf(", reply `%q`", reply)
	}
	return line + " → `" + c.Want[len(c.Want)-1] + "`"
}

func TestVerifiedRunNamesATrace(t *testing.T) {
	for _, p := range Catalog() {
		require.NotEmpty(t, filepath.Base(p.Name))
		require.Contains(t, verifiedRun(p), p.Cases[0].Want[len(p.Cases[0].Want)-1])
	}
}

const docIntro = `# Workflow Design Patterns

Each pattern below is written twice: with the Go ADK API (` + "`google.golang.org/adk/v2/workflow`" + `) and as a declarative ` + "`adk.declarative/v0.2`" + ` document (see [Declarative workflow agents](workflow-agents.md)). The code is not illustrative: it lives in [` + "`go/adk/examples/workflowpatterns`" + `](../../go/adk/examples/workflowpatterns), the tests run both versions on the same inputs and require the same events, and a test fails when this page is out of date.

Run any pattern, in either form or both, from ` + "`go/`" + `:

` + "```sh" + `
go run ./adk/examples/workflowpatterns -list
go run ./adk/examples/workflowpatterns -pattern router -input "bug: crash"
go run ./adk/examples/workflowpatterns -pattern human_approval -input "ship it" -reply yes
go run ./adk/examples/workflowpatterns -pattern fan_in -input Go -mode yaml
go test ./adk/examples/workflowpatterns
` + "```" + `

The models are the offline stub, so nothing needs credentials.
`

const docOutro = `
## Reading the table

- **One result.** A workflow ends in exactly one terminal output. Branches that run concurrently must converge in a ` + "`join`" + `, or all but one must finish without output.
- **Routes are strings.** ` + "`BoolRoute(true)`" + ` and ` + "`IntRoute(1)`" + ` match the strings ` + "`\"true\"`" + ` and ` + "`\"1\"`" + `, which is how YAML writes them. ` + "`DEFAULT`" + ` is the ADK ` + "`Default`" + ` route.
- **Plain steps are silent.** A function node emits data for its successors. The terminal result is also rendered as message text, so A2A clients display the answer and not every intermediate step.
- **Names.** A sub-workflow node cannot share the root agent's name, and an agent node is named after its agent.
- **Where YAML stops.** YAML names behavior; it cannot contain it. Functions, tools, and models come from a ` + "`Registry`" + `, and a custom node type is Go code registered once.
`
