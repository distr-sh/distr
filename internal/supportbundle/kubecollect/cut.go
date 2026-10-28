package kubecollect

import (
	"bytes"
	"fmt"
)

// keepTail cuts content to at most maxBytes from its end, starting at a line, so a log keeps its
// newest lines.
func keepTail(content []byte, maxBytes int) ([]byte, bool) {
	if len(content) <= maxBytes {
		return content, false
	}
	note := fmt.Appendf(nil, "--- distr: earlier output truncated at %d bytes ---\n", maxBytes)
	tail := content[len(content)-max(maxBytes-len(note), 0):]
	if i := bytes.IndexByte(tail, '\n'); i >= 0 && i < len(tail)-1 {
		tail = tail[i+1:]
	}
	return append(note, tail...), true
}

// keepHead cuts content to at most maxBytes from its start. What it cuts is no longer valid YAML or
// JSON, which the comment at the top says.
func keepHead(content []byte, maxBytes int) ([]byte, bool) {
	if len(content) <= maxBytes {
		return content, false
	}
	note := fmt.Appendf(nil, "# --- distr: output truncated at %d bytes ---\n", maxBytes)
	return append(note, content[:max(maxBytes-len(note), 0)]...), true
}
