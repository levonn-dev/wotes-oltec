package emit

import "strings"

// Depfile renders a Makefile-style dependency line, the form Ninja reads back through DEPFILE.
func Depfile(target string, deps []string) string {
	esc := strings.NewReplacer(" ", `\ `, "$", "$$", "#", `\#`)
	parts := make([]string, len(deps))
	for i, d := range deps {
		parts[i] = esc.Replace(d)
	}
	return esc.Replace(target) + ": " + strings.Join(parts, " ") + "\n"
}
