package api

import "encoding/json"

// Nullable tracks whether a JSON field was present (even if null).
// Present=false means the field was omitted; Present=true with Value=nil means explicit null.
type Nullable[T any] struct {
	Value   *T
	Present bool
}

func (n *Nullable[T]) UnmarshalJSON(data []byte) error {
	n.Present = true
	if string(data) == "null" {
		n.Value = nil
		return nil
	}
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	n.Value = &v
	return nil
}
