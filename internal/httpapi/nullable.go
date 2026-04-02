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
