package util

import (
	"os"
	"path"
	"strings"
)

// SumFileSizesWithPrefix returns the total size of the files in the directory of prefixedPath
// whose name starts with its base name. Files that disappear between listing and stat are
// skipped, since a partial sum is more useful than none.
//
// The directory is listed rather than globbed because a caller's path may contain glob
// metacharacters, which would have to be escaped to be matched literally.
func SumFileSizesWithPrefix(prefixedPath string) (int64, error) {
	entries, err := os.ReadDir(path.Dir(prefixedPath))
	if err != nil {
		return 0, err
	}

	prefix := path.Base(prefixedPath)
	var total int64
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), prefix) {
			continue
		}
		if info, err := entry.Info(); err == nil {
			total += info.Size()
		}
	}
	return total, nil
}
