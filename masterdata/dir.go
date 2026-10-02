package masterdata

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

type Failure struct {
	File string
	Err  string
}

type Result struct {
	Done   int
	Failed []Failure
}

func listByExt(dir string, ext string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() && strings.EqualFold(filepath.Ext(e.Name()), ext) {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out, nil
}

func parallel(names []string, workers int, fn func(name string) error) *Result {
	res := &Result{}
	var mu sync.Mutex
	var wg sync.WaitGroup
	queue := make(chan string)
	for w := 0; w < max(workers, 1); w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for n := range queue {
				err := fn(n)
				mu.Lock()
				if err != nil {
					res.Failed = append(res.Failed, Failure{File: n, Err: err.Error()})
				} else {
					res.Done++
				}
				mu.Unlock()
			}
		}()
	}
	for _, n := range names {
		queue <- n
	}
	close(queue)
	wg.Wait()
	sort.Slice(res.Failed, func(i, j int) bool { return res.Failed[i].File < res.Failed[j].File })
	return res
}

func DecodeDir(inDir string, outDir string, k *Key, pretty bool, workers int) (*Result, error) {
	names, err := listByExt(inDir, ".bin")
	if err != nil {
		return nil, err
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("no .bin files in %s", inDir)
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return nil, err
	}
	return parallel(names, workers, func(name string) error {
		raw, err := os.ReadFile(filepath.Join(inDir, name))
		if err != nil {
			return err
		}
		text, err := Decode(raw, k)
		if err != nil {
			return err
		}
		if pretty {
			if text, err = Pretty(text); err != nil {
				return err
			}
		}
		stem := strings.TrimSuffix(name, filepath.Ext(name))
		return os.WriteFile(filepath.Join(outDir, stem+".json"), text, 0644)
	}), nil
}

func EncodeDir(inDir string, outDir string, k *Key, prefixDir string, workers int) (*Result, error) {
	names, err := listByExt(inDir, ".json")
	if err != nil {
		return nil, err
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("no .json files in %s", inDir)
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return nil, err
	}
	return parallel(names, workers, func(name string) error {
		text, err := os.ReadFile(filepath.Join(inDir, name))
		if err != nil {
			return err
		}
		if text, err = Compact(text); err != nil {
			return ErrBadJSON
		}
		stem := strings.TrimSuffix(name, filepath.Ext(name))
		var prefix []byte
		if prefixDir != "" {
			if old, err := os.ReadFile(filepath.Join(prefixDir, stem+".bin")); err == nil && len(old) >= PrefixSize {
				prefix = old[:PrefixSize]
			}
		}
		raw, err := Encode(text, k, prefix)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(outDir, stem+".bin"), raw, 0644)
	}), nil
}
