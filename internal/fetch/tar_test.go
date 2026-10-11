package fetch

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"testing"
)

type entry struct {
	name, body, link string
	typ              byte
	mode             int64
}

// tarball builds a gzipped tar in memory, the same shape GitHub serves.
func tarball(t *testing.T, entries []entry) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for _, e := range entries {
		typ := e.typ
		if typ == 0 {
			typ = tar.TypeReg
		}
		mode := e.mode
		if mode == 0 {
			mode = 0o644
		}
		h := &tar.Header{Name: e.name, Typeflag: typ, Mode: mode, Linkname: e.link, Size: int64(len(e.body))}
		if typ != tar.TypeReg {
			h.Size = 0
		}
		if typ == tar.TypeXGlobalHeader {
			// GitHub tarballs start with one of these; it may only carry PAX records.
			h = &tar.Header{Name: e.name, Typeflag: typ, PAXRecords: map[string]string{"comment": "abc123"}}
		}
		if err := tw.WriteHeader(h); err != nil {
			t.Fatal(err)
		}
		if typ == tar.TypeReg {
			if _, err := tw.Write([]byte(e.body)); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return &buf
}

func paths(r Result) map[string]File {
	m := map[string]File{}
	for _, f := range r.Files {
		m[f.Path] = f
	}
	return m
}

func TestExtractSubdir(t *testing.T) {
	buf := tarball(t, []entry{
		{name: "tmpl-abc123/", typ: tar.TypeDir},
		{name: "tmpl-abc123/README.md", body: "root"},
		{name: "tmpl-abc123/templates/pierre/package.json", body: `{"name":"{{name}}"}`},
		{name: "tmpl-abc123/templates/pierre/scripts/run.sh", body: "#!/bin/sh", mode: 0o755},
		{name: "tmpl-abc123/templates/pierre/node_modules/x/index.js", body: "x"},
		{name: "tmpl-abc123/templates/pierre-old/package.json", body: "old"},
		{name: "tmpl-abc123/templates/pierre/link", typ: tar.TypeSymlink, link: "package.json"},
	})
	res, err := ExtractTarGz(buf, "templates/pierre")
	if err != nil {
		t.Fatal(err)
	}
	got := paths(res)
	if len(got) != 3 {
		t.Fatalf("got %d files: %v", len(got), got)
	}
	if string(got["package.json"].Data) != `{"name":"{{name}}"}` {
		t.Errorf("package.json = %q", got["package.json"].Data)
	}
	if got["scripts/run.sh"].Mode != 0o755 {
		t.Errorf("run.sh mode = %v", got["scripts/run.sh"].Mode)
	}
	if got["link"].LinkTarget != "package.json" {
		t.Errorf("link = %+v", got["link"])
	}
}

func TestExtractWholeRepo(t *testing.T) {
	buf := tarball(t, []entry{
		{name: "pax_global_header", typ: tar.TypeXGlobalHeader},
		{name: "repo-sha/a.txt", body: "a"},
		{name: "repo-sha/.git/config", body: "x"},
	})
	res, err := ExtractTarGz(buf, "")
	if err != nil {
		t.Fatal(err)
	}
	if got := paths(res); len(got) != 1 || got["a.txt"].Path == "" {
		t.Errorf("got %v", got)
	}
}

func TestExtractRejectsUnsafeEntries(t *testing.T) {
	buf := tarball(t, []entry{
		{name: "repo-sha/ok.txt", body: "ok"},
		{name: "repo-sha/../../evil.txt", body: "evil"},
		{name: "repo-sha/escape", typ: tar.TypeSymlink, link: "../../etc/passwd"},
		{name: "repo-sha/abs", typ: tar.TypeSymlink, link: "/etc/passwd"},
		{name: "repo-sha/fifo", typ: tar.TypeFifo},
	})
	res, err := ExtractTarGz(buf, "")
	if err != nil {
		t.Fatal(err)
	}
	if got := paths(res); len(got) != 1 {
		t.Errorf("expected only ok.txt, got %v", got)
	}
	if len(res.Warnings) != 4 {
		t.Errorf("expected 4 warnings, got %v", res.Warnings)
	}
}

func TestExtractEmptySubdirIsNotFound(t *testing.T) {
	buf := tarball(t, []entry{{name: "repo-sha/a.txt", body: "a"}})
	if _, err := ExtractTarGz(buf, "templates/nope"); !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}
