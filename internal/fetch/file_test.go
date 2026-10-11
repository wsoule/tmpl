package fetch

import "testing"

func TestIgnored(t *testing.T) {
	for _, p := range []string{".git", ".git/config", "node_modules", "a/node_modules/x/y.js", ".DS_Store", "src/.DS_Store"} {
		if !Ignored(p) {
			t.Errorf("Ignored(%q) = false", p)
		}
	}
	for _, p := range []string{"src/main.ts", ".github/workflows/ci.yml", ".gitignore", "tmpl.json"} {
		if Ignored(p) {
			t.Errorf("Ignored(%q) = true", p)
		}
	}
}

func TestCleanRel(t *testing.T) {
	ok := map[string]string{"a/b": "a/b", "a/./b": "a/b", "a/../b": "b"}
	for in, want := range ok {
		if got, good := cleanRel(in); !good || got != want {
			t.Errorf("cleanRel(%q) = %q,%v want %q,true", in, got, good, want)
		}
	}
	for _, in := range []string{"", ".", "..", "../x", "a/../../x", "/etc/passwd"} {
		if _, good := cleanRel(in); good {
			t.Errorf("cleanRel(%q) should be rejected", in)
		}
	}
}

func TestLinkStaysInside(t *testing.T) {
	cases := []struct {
		rel, target string
		want        bool
	}{
		{"a/link", "b", true},
		{"a/link", "../b", true},
		{"a/link", "../../b", false},
		{"link", "/etc/passwd", false},
	}
	for _, c := range cases {
		if got := linkStaysInside(c.rel, c.target); got != c.want {
			t.Errorf("linkStaysInside(%q,%q) = %v", c.rel, c.target, got)
		}
	}
}

func TestLinkStaysInsideRejectsDotDotAfterName(t *testing.T) {
	for _, target := range []string{"d/s1/../../z", "a/../b", "./x/../y"} {
		if linkStaysInside("link", target) {
			t.Errorf("linkStaysInside(link, %q) = true", target)
		}
	}
	for _, target := range []string{"x", "./x", "x/y"} {
		if !linkStaysInside("link", target) {
			t.Errorf("linkStaysInside(link, %q) = false", target)
		}
	}
	if !linkStaysInside("a/b/link", "../../x/y") {
		t.Error("leading .. rejected")
	}
}
