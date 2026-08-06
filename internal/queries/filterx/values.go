package filterx

// StringValues converts a caller-supplied slice of a string-kinded API type
// (a generated enum, a typed string) into the []any values a Filter expects.
// Returns nil for a nil input, which parseFilter-style callers treat as "no
// filter supplied".
func StringValues[T ~string](values *[]T) []any {
	if values == nil {
		return nil
	}
	out := make([]any, len(*values))
	for i, v := range *values {
		out[i] = string(v)
	}
	return out
}

// Int64Values converts a caller-supplied slice of an int64-kinded API type
// (typically a typed ID) into the []any values a Filter expects. Returns nil
// for a nil input.
func Int64Values[T ~int64](values *[]T) []any {
	if values == nil {
		return nil
	}
	out := make([]any, len(*values))
	for i, v := range *values {
		out[i] = int64(v)
	}
	return out
}

// ParseFilter resolves a value column's operator and decides whether it
// contributes a filter. A present operator with no values (and no null
// check) is treated as no filter; null operators apply with no values.
func ParseFilter[T ~string](column string, values []any, opPtr *T) (Filter, bool, error) {
	op := OpIn
	if opPtr != nil {
		parsed, err := ParseOperator(string(*opPtr))
		if err != nil {
			return Filter{}, false, err
		}
		op = parsed
	}
	nullCheck := op == OpIsNull || op == OpNotNull
	if len(values) == 0 && !nullCheck {
		return Filter{}, false, nil
	}
	return Filter{Column: column, Op: op, Values: values}, true, nil
}
