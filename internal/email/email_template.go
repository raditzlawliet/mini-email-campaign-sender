package email

import "strings"

// Render replaces {key} placeholders in the template string with values from data.
// It uses strings.NewReplacer for efficient single-pass replacement.
func Render(template string, data map[string]string) string {
	pairs := make([]string, 0, len(data)*2)
	for k, v := range data {
		pairs = append(pairs, "{"+k+"}", v)
	}
	replacer := strings.NewReplacer(pairs...)
	return replacer.Replace(template)
}
