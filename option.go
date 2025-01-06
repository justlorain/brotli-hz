package brotli_hz

import (
	"regexp"
	"strings"
)

type (
	ExcludedPaths       []string
	ExcludedPathRegexes []*regexp.Regexp
	ExcludedExtensions  map[string]struct{}
)

func NewExcludedPaths(paths []string) ExcludedPaths {
	return ExcludedPaths(paths)
}

func (eps ExcludedPaths) Contains(uri string) bool {
	for _, p := range eps {
		if strings.HasPrefix(uri, p) {
			return true
		}
	}
	return false
}

func NewExcludedPathRegexes(regexes []string) ExcludedPathRegexes {
	res := make(ExcludedPathRegexes, len(regexes))
	for i, r := range regexes {
		res[i] = regexp.MustCompile(r)
	}
	return res
}

func (epr ExcludedPathRegexes) Contains(uri string) bool {
	for _, r := range epr {
		if r.MatchString(uri) {
			return true
		}
	}
	return false
}

func NewExcludedExtensions(exts []string) ExcludedExtensions {
	res := make(ExcludedExtensions)
	for _, e := range exts {
		res[e] = struct{}{}
	}
	return res
}

func (ees ExcludedExtensions) Contains(ext string) bool {
	_, ok := ees[ext]
	return ok
}
