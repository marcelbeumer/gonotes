package note

import (
	"regexp"
	"strings"
)

func ParseTags(str string) []string {
	r := regexp.MustCompile("[,\\s]")
	parts := r.Split(str, -1)
	tags := make([]string, 0)
	for _, v := range parts {
		trimmed := strings.TrimSpace(v)
		if trimmed != "" {
			tags = append(tags, v)
		}
	}
	return tags
}
