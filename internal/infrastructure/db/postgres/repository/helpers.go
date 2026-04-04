package repository

import "strings"

// escapeILIKE escapes SQL ILIKE/LIKE special characters (%, _) in user input
// to prevent unintended wildcard matching.
func escapeILIKE(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return r.Replace(s)
}
