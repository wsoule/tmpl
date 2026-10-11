// Package write puts fetched template files on disk without clobbering
// anything the user already has, unless forced.
package write

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/wsoule/tmpl/internal/fetch"
)

// ManifestFile is the template's own metadata; it is never written into projects.
const ManifestFile = "tmpl.json"

// Report lists what happened to each template file.
type Report struct {
	Written []string // paths created or overwritten
	Kept    []string // paths that already existed and were left alone
}

// Files writes files under dest, creating dest if needed. Existing paths are
// kept unless force is set, in which case they are replaced. A directory in
// the way of a file is an error even with force, so nothing is deleted recursively.
//
// Every operation goes through an os.Root opened on dest, so a path that
// resolves outside dest (for example through a chain of template symlinks,
// or a symlink already in dest) is refused instead of written.
func Files(dest string, files []fetch.File, force bool) (Report, error) {
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return Report{}, err
	}
	root, err := os.OpenRoot(dest)
	if err != nil {
		return Report{}, err
	}
	defer root.Close()

	var rep Report
	var links []fs.FileInfo // symlinks written in this run
	for _, f := range files {
		if f.Path == ManifestFile {
			continue
		}
		link, err := createdLinkAncestor(root, f.Path, links)
		if err != nil {
			return rep, fmt.Errorf("write %s: %w", f.Path, err)
		}
		if link != "" {
			return rep, fmt.Errorf("write %s: path is inside symlink %s", f.Path, link)
		}
		name := filepath.FromSlash(f.Path)
		_, err = root.Lstat(name)
		switch {
		case err == nil:
			if !force {
				rep.Kept = append(rep.Kept, f.Path)
				continue
			}
			if err := root.Remove(name); err != nil {
				return rep, fmt.Errorf("replace %s: %w", f.Path, err)
			}
		case !errors.Is(err, fs.ErrNotExist):
			return rep, fmt.Errorf("write %s: %w", f.Path, err)
		}
		if dir := filepath.Dir(name); dir != "." {
			if err := root.MkdirAll(dir, 0o755); err != nil {
				return rep, fmt.Errorf("write %s: %w", f.Path, err)
			}
		}
		if f.LinkTarget != "" {
			if err := root.Symlink(filepath.FromSlash(f.LinkTarget), name); err != nil {
				return rep, fmt.Errorf("write %s: %w", f.Path, err)
			}
			fi, err := root.Lstat(name)
			if err != nil {
				return rep, fmt.Errorf("write %s: %w", f.Path, err)
			}
			links = append(links, fi)
		} else {
			mode := f.Mode
			if mode == 0 {
				mode = 0o644
			}
			if err := root.WriteFile(name, f.Data, mode); err != nil {
				return rep, fmt.Errorf("write %s: %w", f.Path, err)
			}
			// WriteFile's mode is masked by the umask; set it exactly.
			if err := root.Chmod(name, mode); err != nil {
				return rep, fmt.Errorf("write %s: %w", f.Path, err)
			}
		}
		rep.Written = append(rep.Written, f.Path)
	}
	return rep, nil
}

// createdLinkAncestor returns the parent of rel that is one of the symlinks
// written earlier in this run, or "". It asks the filesystem rather than
// comparing names, so case-insensitive or normalizing filesystems (where
// "a/b" may be the link "A/b") can't route a file through such a link.
// os.Root stops writes that escape dest, but not the creation of a link whose
// target escapes, so nothing is ever placed beneath a template's own link.
func createdLinkAncestor(root *os.Root, rel string, links []fs.FileInfo) (string, error) {
	if len(links) == 0 {
		return "", nil
	}
	for i := 0; i < len(rel); i++ {
		if rel[i] != '/' {
			continue
		}
		fi, err := root.Lstat(filepath.FromSlash(rel[:i]))
		if errors.Is(err, fs.ErrNotExist) {
			return "", nil // deeper parents can't exist either
		}
		if err != nil {
			return "", err
		}
		if fi.Mode()&fs.ModeSymlink == 0 {
			continue
		}
		for _, l := range links {
			if os.SameFile(fi, l) {
				return rel[:i], nil
			}
		}
	}
	return "", nil
}
