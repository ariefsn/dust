package cleaner

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// DirSize walks `root` and returns the total byte size and file count.
// Returns (0, 0, nil) if the path does not exist (treated as "nothing to clean").
// Honors ctx cancellation.
func DirSize(ctx context.Context, root string) (int64, int, error) {
	var bytes int64
	var items int

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			if errors.Is(err, fs.ErrPermission) {
				return nil
			}
			return err
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			return err
		}
		bytes += info.Size()
		items++
		return nil
	})

	if err != nil && errors.Is(err, fs.ErrNotExist) {
		return 0, 0, nil
	}
	return bytes, items, err
}

var ErrUnsafePath = errors.New("refusing to delete unsafe path")

// SafeRemoveAll deletes path only if it lives inside one of the allowed roots
// and is not equal to any of them. Treats a non-existent path as a no-op.
func SafeRemoveAll(path string, allowedRoots []string) error {
	if path == "" || path == "/" {
		return fmt.Errorf("%w: %q", ErrUnsafePath, path)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	clean := filepath.Clean(abs)

	home := Home()
	if home != "" && clean == filepath.Clean(home) {
		return fmt.Errorf("%w: %q is $HOME", ErrUnsafePath, path)
	}

	allowed := false
	for _, r := range allowedRoots {
		rAbs, err := filepath.Abs(Expand(r))
		if err != nil {
			continue
		}
		rClean := filepath.Clean(rAbs)
		if clean == rClean {
			return fmt.Errorf("%w: %q equals an allowed root", ErrUnsafePath, path)
		}
		if hasPrefixDir(clean, rClean) {
			allowed = true
			break
		}
	}
	if !allowed {
		return fmt.Errorf("%w: %q is not under any allowed root", ErrUnsafePath, path)
	}

	if _, err := os.Stat(clean); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	return os.RemoveAll(clean)
}

// hasPrefixDir reports whether `p` is `prefix` or any descendant of `prefix`.
func hasPrefixDir(p, prefix string) bool {
	if p == prefix {
		return true
	}
	prefix = filepath.Clean(prefix)
	p = filepath.Clean(p)
	if len(p) <= len(prefix) {
		return false
	}
	if p[:len(prefix)] != prefix {
		return false
	}
	return p[len(prefix)] == filepath.Separator
}
