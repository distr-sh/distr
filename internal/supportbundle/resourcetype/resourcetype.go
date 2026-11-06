// Package resourcetype is shared by Distr and the support collector, which runs at customers and
// must not link the database models of internal/types.
package resourcetype

import (
	"slices"
	"strings"

	"github.com/distr-sh/distr/internal/contenttype"
)

var ContentTypes = []string{
	contenttype.MediaTypeTextPlain,
	contenttype.MediaTypeTextXLog,
	contenttype.MediaTypeYAML,
	contenttype.MediaTypeJSON,
}

func ParseContentType(value string) string {
	if slices.Contains(ContentTypes, value) {
		return value
	}
	return contenttype.MediaTypeTextPlain
}

func FileExtension(contentType string) string {
	switch contentType {
	case contenttype.MediaTypeTextXLog:
		return ".log"
	case contenttype.MediaTypeYAML:
		return ".yaml"
	case contenttype.MediaTypeJSON:
		return ".json"
	default:
		return ".txt"
	}
}

// ContentTypeOfFile is the content type whose FileExtension the name ends with, or plain text.
func ContentTypeOfFile(name string) string {
	for _, contentType := range ContentTypes {
		if contentType != contenttype.MediaTypeTextPlain && strings.HasSuffix(name, FileExtension(contentType)) {
			return contentType
		}
	}
	return contenttype.MediaTypeTextPlain
}
