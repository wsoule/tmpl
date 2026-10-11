package rename

import (
	"reflect"
	"testing"

	"github.com/wsoule/tmpl/internal/fetch"
)

func TestValidate(t *testing.T) {
	for _, ok := range []string{"dogear", "my-app", "a.b_c", "9lives"} {
		if err := Validate(ok); err != nil {
			t.Errorf("Validate(%q) = %v", ok, err)
		}
	}
	for _, bad := range []string{"", "Dogear", "my app", "-x", ".x", "a/b", "@scope/x"} {
		if err := Validate(bad); err == nil {
			t.Errorf("Validate(%q) expected error", bad)
		}
	}
}

func TestSuggest(t *testing.T) {
	cases := map[string]string{
		"Dogear":    "dogear",
		"My App":    "my-app",
		"--weird--": "weird--",
		"émoji✨app": "moji-app",
		"!!!":       "app",
		"":          "app",
	}
	for in, want := range cases {
		got := Suggest(in)
		if got != want {
			t.Errorf("Suggest(%q) = %q, want %q", in, got, want)
		}
		if err := Validate(got); err != nil {
			t.Errorf("Suggest(%q) = %q is invalid: %v", in, got, err)
		}
	}
}

func TestIsText(t *testing.T) {
	if !IsText([]byte("hello {{name}}")) {
		t.Error("plain text reported binary")
	}
	if IsText([]byte{0x89, 'P', 'N', 'G', 0x00, 0x01}) {
		t.Error("binary reported text")
	}
}

func TestApply(t *testing.T) {
	files := []fetch.File{
		{Path: "package.json", Data: []byte(`{"name":"{{name}}"}`)},
		{Path: "README.md", Data: []byte("# {{name}}\n\n{{name}} is great")},
		{Path: "logo.png", Data: []byte("\x00{{name}}")},
		{Path: "plain.txt", Data: []byte("nothing here")},
		{Path: "link", LinkTarget: "{{name}}"},
	}
	if !Uses(files) {
		t.Fatal("Uses = false")
	}
	changed := Apply(files, "dogear")
	if !reflect.DeepEqual(changed, []string{"package.json", "README.md"}) {
		t.Errorf("changed = %v", changed)
	}
	if string(files[0].Data) != `{"name":"dogear"}` {
		t.Errorf("package.json = %s", files[0].Data)
	}
	if string(files[1].Data) != "# dogear\n\ndogear is great" {
		t.Errorf("README = %s", files[1].Data)
	}
	if string(files[2].Data) != "\x00{{name}}" {
		t.Error("binary file was modified")
	}
	if files[4].LinkTarget != "{{name}}" {
		t.Error("symlink was modified")
	}
}

func TestUsesFalse(t *testing.T) {
	if Uses([]fetch.File{{Path: "a", Data: []byte("x")}}) {
		t.Error("Uses = true")
	}
}
