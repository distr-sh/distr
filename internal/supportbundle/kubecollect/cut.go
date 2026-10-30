package kubecollect

import (
	"bytes"
	"errors"
	"fmt"
	"io"
)

// keepTail reads r to its end and keeps at most its last maxBytes, starting at a line, so a log
// keeps its newest lines. It holds at most twice maxBytes in memory.
func keepTail(r io.Reader, maxBytes int) ([]byte, error) {
	var content []byte
	cut := false
	chunk := make([]byte, 32*1024)
	for {
		n, err := r.Read(chunk)
		content = append(content, chunk[:n]...)
		if len(content) > 2*maxBytes {
			content = append(content[:0], content[len(content)-maxBytes:]...)
			cut = true
		}
		if errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return nil, err
		}
	}
	if !cut && len(content) <= maxBytes {
		return content, nil
	}
	note := fmt.Appendf(nil, "--- distr: earlier output truncated at %d bytes ---\n", maxBytes)
	tail := content[len(content)-min(len(content), max(maxBytes-len(note), 0)):]
	if i := bytes.IndexByte(tail, '\n'); i >= 0 && i < len(tail)-1 {
		tail = tail[i+1:]
	}
	return append(note, tail...), nil
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
