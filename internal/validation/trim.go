package validation

import (
	"reflect"
	"strings"
	"sync"
)

// TrimStrings trims leading and trailing whitespace from every string reachable from v, which must
// be a pointer. Only a field tagged `trim:"-"` is left untouched.
func TrimStrings(v any) {
	value := reflect.ValueOf(v)
	if value.Kind() != reflect.Pointer {
		return
	}
	trimValue(value.Elem())
}

func trimValue(value reflect.Value) {
	if !canContainString(value.Type()) {
		return
	}
	switch value.Kind() {
	case reflect.String:
		if current := value.String(); value.CanSet() {
			if trimmed := strings.TrimSpace(current); trimmed != current {
				value.SetString(trimmed)
			}
		}
	case reflect.Pointer, reflect.Interface:
		if !value.IsNil() {
			trimValue(value.Elem())
		}
	case reflect.Slice, reflect.Array:
		for i := range value.Len() {
			trimValue(value.Index(i))
		}
	case reflect.Map:
		// A map value is not addressable, so it has to be trimmed in a copy and written back.
		for _, key := range value.MapKeys() {
			element := reflect.New(value.Type().Elem()).Elem()
			element.Set(value.MapIndex(key))
			trimValue(element)
			value.SetMapIndex(key, element)
		}
	case reflect.Struct:
		for i := range value.NumField() {
			if field := value.Type().Field(i); field.IsExported() && !isTrimExcluded(field) {
				trimValue(value.Field(i))
			}
		}
	}
}

func isTrimExcluded(field reflect.StructField) bool {
	return field.Tag.Get("trim") == "-"
}

var canContainStringCache sync.Map

// canContainString keeps TrimStrings from walking types that hold no string at all, most importantly
// the []byte payloads (compose files, values.yaml, env files) that would otherwise be visited byte
// by byte.
func canContainString(t reflect.Type) bool {
	if cached, ok := canContainStringCache.Load(t); ok {
		return cached.(bool)
	}
	result := computeCanContainString(t, map[reflect.Type]bool{})
	canContainStringCache.Store(t, result)
	return result
}

func computeCanContainString(t reflect.Type, seen map[reflect.Type]bool) bool {
	if seen[t] {
		return false
	}
	seen[t] = true
	switch t.Kind() {
	case reflect.String, reflect.Interface:
		return true
	case reflect.Pointer, reflect.Slice, reflect.Array, reflect.Map:
		return computeCanContainString(t.Elem(), seen)
	case reflect.Struct:
		for i := range t.NumField() {
			field := t.Field(i)
			if field.IsExported() && !isTrimExcluded(field) && computeCanContainString(field.Type, seen) {
				return true
			}
		}
		return false
	default:
		return false
	}
}
