package httpapi

import "encoding/json"

// NullableString distinguishes three JSON states for a string field:
//   - absent from JSON body → Set=false, Value=nil  (don't change)
//   - explicit null in JSON → Set=true,  Value=nil  (clear)
//   - string value in JSON  → Set=true,  Value=&s   (set new value)
//
// Use x-go-type: NullableString in openapi.yaml for request body fields
// that need this distinction. The standard *string cannot represent "absent"
// separately from "null" because both produce a nil pointer after decoding.
type NullableString struct {
	Value *string
	Set   bool
}

// MarshalJSON emits the string value or null. The zero value (Set=false) also
// marshals as null: Go's encoding/json cannot omit a non-pointer struct field
// from a parent object, so callers that need field-level "absent" semantics
// must send raw JSON instead of using the typed struct.
func (n NullableString) MarshalJSON() ([]byte, error) {
	if n.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(*n.Value)
}

func (n *NullableString) UnmarshalJSON(data []byte) error {
	n.Set = true
	if string(data) == "null" {
		n.Value = nil
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	n.Value = &s
	return nil
}
