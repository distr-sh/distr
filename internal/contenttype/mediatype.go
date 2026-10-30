package contenttype

import (
	"fmt"
	"net/textproto"
	"strings"
)

const (
	MediaTypeJSON             = "application/json"
	MediaTypeYAML             = "application/yaml"
	MediaTypeZIP              = "application/zip"
	MediaTypeOctetStream      = "application/octet-stream"
	MediaTypeTextYAML         = "text/yaml"
	MediaTypeTextPlain        = "text/plain"
	MediaTypeTextXLog         = "text/x-log"
	MediaTypeTextXYaml        = "text/x-yaml"
	MediaTypeApplicationXYaml = "application/x-yaml"
)

func IsYaml(header textproto.MIMEHeader) error {
	return HasMediaType(header,
		MediaTypeJSON,
		MediaTypeYAML,
		MediaTypeTextYAML,
		MediaTypeTextXYaml,
		MediaTypeApplicationXYaml)
}

func HasMediaType(header textproto.MIMEHeader, acceptedContentTypes ...string) error {
	contentType, err := ParseContentType(header.Get("Content-Type"))
	if err != nil {
		return err
	}
	if contentType.MediaType == "" {
		return nil
	}
	for _, t := range acceptedContentTypes {
		if contentType.MediaType == t {
			return nil
		}
	}
	return fmt.Errorf("unacceptable media type: %v (acceptable media types are %v)",
		contentType.MediaType, strings.Join(acceptedContentTypes, ", "))
}
