package gonotes

import (
	"regexp"
	"strings"
)

var (
	reDisallowedSlugChars = regexp.MustCompile("[^a-z0-9-]")
	reMultiDash           = regexp.MustCompile("-{2,}")
)

func slugify(v string) string {
	res := strings.TrimSpace(strings.ToLower(v))
	res = reDisallowedSlugChars.ReplaceAllString(res, "-")
	res = reMultiDash.ReplaceAllString(res, "-")
	res = strings.Trim(res, "-")
	return res
}
