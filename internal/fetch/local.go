package fetch

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Local reads a template directory from disk, applying the same skip and
// symlink rules as tarball extraction.
func Local(root string) (Result, error) {
	fi, err := os.Stat(root)
	if err != nil || !fi.IsDir() {
		return Result{}, fmt.Errorf("%w: %s", ErrNotFound, root)
	}
	var res Result
	err = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if p == root {
			return nil
		}
		relOS, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		rel := filepath.ToSlash(relOS)
		if Ignored(rel) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		switch {
		case d.IsDir():
			return nil
		case d.Type()&fs.ModeSymlink != 0:
			target, err := os.Readlink(p)
			if err != nil {
				return err
			}
			target = filepath.ToSlash(target)
			if !linkStaysInside(rel, target) {
				res.Warnings = append(res.Warnings, fmt.Sprintf("skipped symlink %s -> %s (points outside the template)", rel, target))
				return nil
			}
			res.Files = append(res.Files, File{Path: rel, Mode: 0o777, LinkTarget: target})
		case d.Type().IsRegular():
			info, err := d.Info()
			if err != nil {
				return err
			}
			data, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			res.Files = append(res.Files, File{Path: rel, Mode: info.Mode().Perm(), Data: data})
		default:
			res.Warnings = append(res.Warnings, fmt.Sprintf("skipped special file %s", rel))
		}
		return nil
	})
	if err != nil {
		return Result{}, fmt.Errorf("read %s: %w", root, err)
	}
	if len(res.Files) == 0 {
		return Result{}, fmt.Errorf("%w: %s is empty", ErrNotFound, root)
	}
	return res, nil
}
