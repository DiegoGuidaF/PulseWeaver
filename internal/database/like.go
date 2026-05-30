package database

import "strings"

// EscapeLIKE escapes SQL LIKE wildcards (`%`, `_`) and the escape character itself
// (`\`) in user-supplied input so it matches literally. Pair the returned value with
// an `ESCAPE '\'` clause in the query, e.g.:
//
//	sq.Expr(`col LIKE ? ESCAPE '\'`, "%"+database.EscapeLIKE(input)+"%")
func EscapeLIKE(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}
