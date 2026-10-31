package snippet

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestSourceIsValidGo(t *testing.T) {
	if _, err := parser.ParseFile(token.NewFileSet(), "snippet.go", Source, parser.ParseComments); err != nil {
		t.Fatalf("Source does not parse: %v", err)
	}
}

func TestSourceCoversSyntax(t *testing.T) {
	want := []string{
		"package", "import (", "const (", "iota", "type", "struct", "`", "interface",
		"[T any]", "func (", "go func", "chan", "select", "defer", "switch", ".(type)",
		"range", "map[", "[]", "make(", "return", "if err != nil", "// ", "/*",
		":=", "<-", "==", "<=", ">=", "&&", "||", "...", "->",
		"", "", "",
	}
	for _, w := range want {
		if !strings.Contains(Source, w) {
			t.Errorf("Source is missing %q", w)
		}
	}
}

func TestHighlightRoundTrip(t *testing.T) {
	var sb strings.Builder
	for _, s := range Highlight(Source) {
		sb.WriteString(s.Text)
	}
	if sb.String() != Source {
		t.Error("concatenated spans differ from Source")
	}
}

func TestHighlightClasses(t *testing.T) {
	spans := Highlight("func main() {\n\tx := 1 // hi\n\ts := \"a\"\n\t_ = nil\n\t_ = true\n\tvar t string\n\ta -> b\n}\n")
	got := map[string]string{}
	for _, s := range spans {
		if _, seen := got[s.Text]; !seen {
			got[s.Text] = s.Class
		}
	}
	want := map[string]string{
		"func": "keyword", "main": "func", ":=": "operator", "1": "number",
		"// hi": "comment", `"a"`: "string", "nil": "builtin", "true": "builtin",
		"string": "builtin", "x": "ident", "{": "punct",
	}
	for text, class := range want {
		if got[text] != class {
			t.Errorf("%q has class %q, want %q", text, got[text], class)
		}
	}
	// Adjacent operator characters stay in one span so ligatures can form.
	if got["->"] != "operator" {
		t.Errorf("-> should be one operator span, got class %q", got["->"])
	}
}
