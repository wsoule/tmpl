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
	links := map[string]bool{} // symlinks written in this run
	for _, f := range files {
		if f.Path == ManifestFile {
			continue
		}
		if link := fetch.LinkAncestor(f.Path, links); link != "" {
			return rep, fmt.Errorf("write %s: path is inside symlink %s", f.Path, link)
		}
		name := filepath.FromSlash(f.Path)
		_, err := root.Lstat(name)
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
			links[f.Path] = true
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
