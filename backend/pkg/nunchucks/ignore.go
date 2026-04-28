package nunchucks

import (
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// TemplateRelIgnored reports whether rel (path relative to views, slash-separated)
// matches any of the ignore patterns. Patterns use doublestar glob syntax (e.g. "**/draft/*.njk").
func TemplateRelIgnored(rel string, patterns []string) bool {
	if len(patterns) == 0 {
		return false
	}
	rel = filepath.ToSlash(rel)
	for _, p := range patterns {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		p = filepath.ToSlash(p)
		ok, err := doublestar.Match(p, rel)
		if err == nil && ok {
			return true
		}
	}
	return false
}
