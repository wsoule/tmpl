package fetch

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"strings"
)

// ExtractTarGz reads a GitHub-style .tar.gz (everything under one top-level
// "<repo>-<sha>/" directory) and returns the files under subdir, with that
// prefix stripped. An empty result is ErrNotFound.
func ExtractTarGz(r io.Reader, subdir string) (Result, error) {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return Result{}, fmt.Errorf("read tarball: %w", err)
	}
	defer gz.Close()

	prefix := ""
	if s := strings.Trim(subdir, "/"); s != "" {
		prefix = s + "/"
	}

	var res Result
	links := map[string]bool{}
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		// With GODEBUG=tarinsecurepath=0, Next reports ".." names as
		// ErrInsecurePath but still returns a valid header; cleanRel below
		// rejects those entries, so keep reading.
		if err != nil && !errors.Is(err, tar.ErrInsecurePath) {
			return Result{}, fmt.Errorf("read tarball: %w", err)
		}
		if h.Typeflag == tar.TypeXGlobalHeader || h.Typeflag == tar.TypeXHeader {
			continue
		}
		i := strings.Index(h.Name, "/")
		if i < 0 {
			continue
		}
		name := h.Name[i+1:]
		if prefix != "" {
			if !strings.HasPrefix(name, prefix) {
				continue
			}
			name = name[len(prefix):]
		}
		if name == "" || h.Typeflag == tar.TypeDir {
			continue
		}
		rel, ok := cleanRel(name)
		if !ok {
			res.Warnings = append(res.Warnings, fmt.Sprintf("skipped unsafe path %q", h.Name))
			continue
		}
		if Ignored(rel) {
			continue
		}
		if link := linkAncestor(rel, links); link != "" {
			res.Warnings = append(res.Warnings, fmt.Sprintf("skipped %s (inside symlink %s)", rel, link))
			continue
		}
		switch h.Typeflag {
		case tar.TypeReg:
			data, err := io.ReadAll(tr)
			if err != nil {
				return Result{}, fmt.Errorf("read %s: %w", rel, err)
			}
			res.Files = append(res.Files, File{Path: rel, Mode: fs.FileMode(h.Mode).Perm(), Data: data})
		case tar.TypeSymlink:
			if !linkStaysInside(rel, h.Linkname) {
				res.Warnings = append(res.Warnings, fmt.Sprintf("skipped symlink %s -> %s (points outside the template)", rel, h.Linkname))
				continue
			}
			links[strings.ToLower(rel)] = true
			res.Files = append(res.Files, File{Path: rel, Mode: 0o777, LinkTarget: h.Linkname})
		default:
			res.Warnings = append(res.Warnings, fmt.Sprintf("skipped special file %s", rel))
		}
	}
	if len(res.Files) == 0 {
		return Result{}, ErrNotFound
	}
	return res, nil
}
