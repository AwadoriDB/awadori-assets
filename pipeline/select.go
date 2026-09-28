package pipeline

import (
	"regexp"

	"awano-winter/catalog"
	"awano-winter/rich"
)

func compile(pattern string) *regexp.Regexp {
	if pattern == "" {
		return nil
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		rich.Panic("Invalid regex pattern %q: %v", pattern, err)
	}
	return re
}

func matches(re *regexp.Regexp, b catalog.Bundle) bool {
	if re.MatchString(b.Name) {
		return true
	}
	for _, k := range b.Keys {
		if re.MatchString(k) {
			return true
		}
	}
	return false
}

func selectBundles(cat *catalog.Catalog, o Options) []catalog.Bundle {
	keys := cat.SortedKeys(o.Prefix)
	if o.Get != "" {
		keys = []string{o.Get}
	}
	bundles, err := cat.Bundles(keys)
	if err != nil {
		rich.PanicError("Failed to resolve bundles.", err)
	}
	include := compile(o.FilterRegex)
	exclude := compile(o.ExcludeRegex)
	if include == nil && exclude == nil {
		return bundles
	}
	var out []catalog.Bundle
	for _, b := range bundles {
		if include != nil && !matches(include, b) {
			continue
		}
		if exclude != nil && matches(exclude, b) {
			continue
		}
		out = append(out, b)
	}
	return out
}
