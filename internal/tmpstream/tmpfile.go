package tmpstream

import (
	"errors"
	"fmt"
	"io"
	"os"
)

type tmpFileStream struct{ fileName string }

// Destroy implements TmpStream.
func (t *tmpFileStream) Destroy() error {
	if err := os.Remove(t.fileName); err != nil {
		return fmt.Errorf("failed to cleanup tempfile: %w", err)
	}
	return nil
}

// Get implements TmpStream.
func (t *tmpFileStream) Get() (ReadSeekAtCloser, error) {
	if file, err := os.Open(t.fileName); err != nil {
		return nil, fmt.Errorf("failed to open tempfile: %w", err)
	} else {
		return file, nil
	}
}

func newTempFileStream(dir string, src io.Reader) (result TmpStream, rerr error) {
	tmpFile, err := os.CreateTemp(dir, "blob")
	if err != nil {
		return nil, fmt.Errorf("failed to create tempfile: %w", err)
	}
	defer func() { rerr = errors.Join(rerr, tmpFile.Close()) }()
	if _, err := io.Copy(tmpFile, src); err != nil {
		return nil, fmt.Errorf("failed to copy source to tempfile: %w", err)
	}
	return &tmpFileStream{fileName: tmpFile.Name()}, nil
}
