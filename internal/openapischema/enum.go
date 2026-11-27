package openapischema

import "github.com/oaswrap/spec/openapi"

// StringEnum returns a new schema on every call, since the reflector modifies the schema a type exposes in place
// (nullability, field tags) and a shared one would carry those changes into every other use of the type.
func StringEnum[T ~string](values ...T) *openapi.Schema {
	enum := make([]any, len(values))
	for i, value := range values {
		enum[i] = string(value)
	}
	return &openapi.Schema{Type: "string", Enum: enum}
}
