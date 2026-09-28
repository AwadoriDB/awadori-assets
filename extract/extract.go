package extract

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"awano-winter/unity"
)

type Result struct {
	Written []string
	Skipped map[int32]int
}

func fileName(name string, pathID int64, used map[string]bool) string {
	name = filepath.Base(strings.ReplaceAll(name, "\\", "/"))
	if name == "" || name == "." || name == ".." {
		name = fmt.Sprintf("asset_%d", pathID)
	}
	if filepath.Ext(name) == "" {
		name += ".bytes"
	}
	if used[name] {
		ext := filepath.Ext(name)
		name = fmt.Sprintf("%s_%d%s", strings.TrimSuffix(name, ext), pathID, ext)
	}
	used[name] = true
	return name
}

func collect(file *unity.File, outDir string, res *Result, used map[string]bool) error {
	for _, o := range file.Objects {
		if o.ClassID != unity.ClassTextAsset {
			res.Skipped[o.ClassID]++
			continue
		}
		name, script, err := file.TextAsset(o)
		if err != nil {
			return err
		}
		dst := filepath.Join(outDir, fileName(name, o.PathID, used))
		if err := os.MkdirAll(outDir, 0755); err != nil {
			return err
		}
		if err := os.WriteFile(dst, script, 0644); err != nil {
			return err
		}
		res.Written = append(res.Written, dst)
	}
	return nil
}

func File(src string, outDir string) (*Result, error) {
	data, err := os.ReadFile(src)
	if err != nil {
		return nil, err
	}
	res := &Result{Skipped: map[int32]int{}}
	used := map[string]bool{}
	if !unity.IsArchive(data) {
		file, err := unity.ParseSerialized(data)
		if err != nil {
			return nil, fmt.Errorf("not a Unity archive or serialized file")
		}
		return res, collect(file, outDir, res, used)
	}
	arc, err := unity.OpenArchive(data)
	if err != nil {
		return nil, err
	}
	for _, n := range arc.Nodes {
		if strings.HasSuffix(n.Path, ".resS") || strings.HasSuffix(n.Path, ".resource") {
			continue
		}
		file, err := unity.ParseSerialized(arc.NodeData(n))
		if err != nil {
			continue
		}
		if err := collect(file, outDir, res, used); err != nil {
			return nil, err
		}
	}
	return res, nil
}
