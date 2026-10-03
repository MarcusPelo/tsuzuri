package highlight_test

import (
	"testing"

	"github.com/jaisuriya-11/tsuzuri/internal/highlight"
	"github.com/jaisuriya-11/tsuzuri/internal/theme"
)

func TestCodeColoursKnownLanguages(t *testing.T) {
	th := theme.DefaultTheme()
	cases := map[string]string{
		"c":          "#include <stdio.h>\nint main(void) { return 0; }",
		"go":         "package main\nfunc main() { fmt.Println(\"hi\") }",
		"python":     "def f(x):\n    return 'a' + str(x)  # c",
		"javascript": "const x = () => { return 42 }",
		"rust":       "fn main() { let s = String::new(); }",
		"bash":       "for i in 1 2; do echo \"$i\"; done",
		"sql":        "SELECT * FROM t WHERE id = 1;",
		"yaml":       "key: value\nlist:\n  - 1",
	}
	for lang, code := range cases {
		cols := highlight.Code(lang, code, th)
		colored := 0
		for _, line := range cols {
			for _, c := range line {
				if c != "" {
					colored++
				}
			}
		}
		if colored == 0 {
			t.Errorf("%s: expected some highlighted runes", lang)
		}
	}
}

func TestMarkdownHighlightsFencedCode(t *testing.T) {
	th := theme.DefaultTheme()
	doc := "# Title\n\n```c\nint x = 1;\n```\n- item `code`"
	cols := highlight.Markdown(doc, th)
	if len(cols) != 6 {
		t.Fatalf("expected 6 lines, got %d", len(cols))
	}
	if cols[0][0] != th.Blue {
		t.Errorf("heading should be blue")
	}
	if cols[3][0] != th.Yellow { // "int" is a type keyword
		t.Errorf("expected C type keyword coloured, got %q", cols[3][0])
	}
	if cols[5][0] != th.Red {
		t.Errorf("list marker should be coloured")
	}
}
