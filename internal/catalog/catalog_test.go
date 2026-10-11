package catalog

import (
	"reflect"
	"testing"

	"github.com/wsoule/tmpl/internal/fetch"
)

func TestList(t *testing.T) {
	files := []fetch.File{
		{Path: "pierre/package.json", Data: []byte("{}")},
		{Path: "pierre/tmpl.json", Data: []byte(`{"description":"Pierre monorepo"}`)},
		{Path: "bun/README.md", Data: []byte("x")},
		{Path: "broken/tmpl.json", Data: []byte(`not json`)},
		{Path: "README.md", Data: []byte("loose file in templates/")},
	}
	want := []Entry{
		{Name: "broken"},
		{Name: "bun"},
		{Name: "pierre", Description: "Pierre monorepo"},
	}
	if got := List(files); !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v", got)
	}
}
