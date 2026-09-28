package pipeline

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"awano-winter/catalog"
	"awano-winter/extract"
	"awano-winter/rich"
)

func report(src string, dst string, res *extract.Result, err error) bool {
	if err != nil {
		rich.Warning("Skipping %q: %v.", filepath.Base(src), err)
		return false
	}
	for _, p := range res.Written {
		rich.Info("%s -> %s", filepath.Base(src), p)
	}
	if len(res.Skipped) > 0 {
		var ids []int
		for id := range res.Skipped {
			ids = append(ids, int(id))
		}
		sort.Ints(ids)
		var parts []string
		for _, id := range ids {
			parts = append(parts, fmt.Sprintf("%dx%d", id, res.Skipped[int32(id)]))
		}
		rich.Info("%s: unsupported objects (classID x count): %s", filepath.Base(src), strings.Join(parts, ", "))
	}
	return true
}

func extractBundles(bundles []catalog.Bundle) {
	extracted := 0
	for _, b := range bundles {
		src := filepath.Join(decryptedDir, b.Name)
		if _, err := os.Stat(src); err != nil {
			continue
		}
		sub := "_shared"
		if len(b.Keys) > 0 {
			sub = path.Dir(b.Keys[0])
		}
		dst := filepath.Join(extractedDir, filepath.FromSlash(sub))
		res, err := extract.File(src, dst)
		if report(src, dst, res, err) {
			extracted += len(res.Written)
		}
	}
	rich.Info("Extracted %d file(s) into %s.", extracted, extractedDir)
}

func UnpackPath(target string) {
	info, err := os.Stat(target)
	if err != nil {
		rich.PanicError("Cannot read unpack target.", err)
	}
	extracted := 0
	one := func(src string) {
		name := strings.TrimSuffix(filepath.Base(src), filepath.Ext(src))
		dst := filepath.Join(extractedDir, name)
		res, err := extract.File(src, dst)
		if report(src, dst, res, err) {
			extracted += len(res.Written)
		}
	}
	if !info.IsDir() {
		one(target)
	} else {
		filepath.WalkDir(target, func(p string, d fs.DirEntry, err error) error {
			if err == nil && !d.IsDir() {
				one(p)
			}
			return nil
		})
	}
	rich.Info("Extracted %d file(s) into %s.", extracted, extractedDir)
}
