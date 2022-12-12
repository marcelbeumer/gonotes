package util

import (
	"regexp"
	"strings"
)

func Slugify(v string) string {
	disallowedChars := regexp.MustCompile("[^a-z0-9-]")
	doubleDash := regexp.MustCompile("-{2,}")
	trailingSlash := regexp.MustCompile("-$")
	leadingSlash := regexp.MustCompile("^-")
	res := strings.ToLower(v)
	res = disallowedChars.ReplaceAllString(res, "-")
	res = doubleDash.ReplaceAllString(res, "-")
	res = trailingSlash.ReplaceAllString(res, "")
	res = leadingSlash.ReplaceAllString(res, "")
	return res
}
