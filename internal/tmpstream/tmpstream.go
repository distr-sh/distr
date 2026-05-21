package tmpstream

import (
	"io"

	"github.com/distr-sh/distr/internal/env"
)

type ReadSeekAt interface {
	io.ReadSeeker
	io.ReaderAt
}

type ReadSeekAtCloser interface {
	ReadSeekAt
	io.Closer
}

// TmpStream represents a resource that can be accessed via io interfaces and destroyed if no longer needed.
type TmpStream interface {
	Get() (ReadSeekAtCloser, error)
	Destroy() error
}

// New creates a new TmpStream that buffers an [io.Reader] into a [ReadSeekAtCloser] (seekable and randomly
// addressable via [io.ReaderAt], e.g. for use with [io.NewSectionReader]) by either buffering it in memory
// or writing it to a temporary file, depending on whether [env.RegistryScratchDir] is set.
func New(src io.Reader) (TmpStream, error) {
	if dir := env.RegistryScratchDir(); dir == nil {
		return newInMemoryStream(src)
	} else {
		return newTempFileStream(*dir, src)
	}
}
