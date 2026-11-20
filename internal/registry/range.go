package registry

import (
	"fmt"
	"strconv"
	"strings"
)

// parseRange returns the first and last byte, both inclusive, that a single range of a Range header selects from a
// blob of the given size. It accepts "bytes=a-b", "bytes=a-" and "bytes=-n" and clamps the end to the last byte, as
// RFC 9110 section 14.1.2 requires. containerd resumes an interrupted download with "bytes=a-".
//
// It reports false for a header that has to be answered with the whole blob, as RFC 9110 section 14.2 allows for
// several ranges and requires for a unit other than bytes. A 416 would claim that no requested range is satisfiable.
func parseRange(header string, size int64) (int64, int64, bool, error) {
	spec, ok := strings.CutPrefix(header, "bytes=")
	if !ok || strings.Contains(spec, ",") {
		return 0, 0, false, nil
	}
	first, last, ok := strings.Cut(spec, "-")
	if !ok {
		return 0, 0, false, fmt.Errorf("invalid range %q", header)
	}

	var start, end int64
	if first == "" {
		if n, err := parseRangeOffset(last); err != nil || n == 0 {
			return 0, 0, false, fmt.Errorf("invalid range %q", header)
		} else {
			start, end = max(size-n, 0), size-1
		}
	} else if s, err := parseRangeOffset(first); err != nil {
		return 0, 0, false, fmt.Errorf("invalid range %q", header)
	} else if last == "" {
		start, end = s, size-1
	} else if e, err := parseRangeOffset(last); err != nil || e < s {
		return 0, 0, false, fmt.Errorf("invalid range %q", header)
	} else {
		start, end = s, min(e, size-1)
	}

	if start >= size {
		return 0, 0, false, fmt.Errorf("range %q is not within size %d", header, size)
	}
	return start, end, true, nil
}

func parseRangeOffset(s string) (int64, error) {
	if s == "" || strings.TrimLeft(s, "0123456789") != "" {
		return 0, fmt.Errorf("invalid range offset %q", s)
	}
	return strconv.ParseInt(s, 10, 64)
}
