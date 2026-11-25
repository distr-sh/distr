package manifest

import "errors"

var (
	ErrNameUnknown      = errors.New("unknown name")
	ErrManifestUnknown  = errors.New("unknown manifest")
	ErrTagAlreadyExists = errors.New("tag already exists")
	ErrTagInvalid       = errors.New("invalid tag")
)
