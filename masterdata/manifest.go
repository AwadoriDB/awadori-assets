package masterdata

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/awadoriproj/awadori-assets/network"
	"github.com/awadoriproj/awadori-assets/rich"
)

const ManifestName = "MasterManifest.json"

type ManifestFile struct {
	Name string `json:"name"`
	Hash string `json:"hash"`
	Size int64  `json:"size"`
}

type Manifest struct {
	Version string         `json:"version"`
	Files   []ManifestFile `json:"files"`
}

func BuildManifest(dir string, version string) (*Manifest, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	m := &Manifest{Version: version, Files: []ManifestFile{}}
	for _, e := range entries {
		if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), ".bin") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		sum := sha256.Sum256(data)
		m.Files = append(m.Files, ManifestFile{Name: e.Name(), Hash: hex.EncodeToString(sum[:]), Size: int64(len(data))})
	}
	sort.Slice(m.Files, func(i, j int) bool { return m.Files[i].Name < m.Files[j].Name })
	return m, nil
}

func WriteManifest(dir string, m *Manifest) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, ManifestName), data, 0644)
}

func ReadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

type DownloadResult struct {
	Downloaded int
	Kept       int
	Failed     []Failure
}

func validName(name string) bool {
	return name != "" && name != "." && name != ".." && !strings.ContainsAny(name, `/\`)
}

func Download(root string, version string, outDir string, workers int) (*DownloadResult, error) {
	base := fmt.Sprintf("%s/master/%s", strings.TrimRight(root, "/"), version)
	raw, err := network.Bytes(base + "/" + ManifestName)
	if err != nil {
		return nil, fmt.Errorf("manifest download failed: %w", err)
	}
	var m Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("manifest is not valid JSON: %w", err)
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return nil, err
	}
	res := &DownloadResult{}
	var mu sync.Mutex
	var wg sync.WaitGroup
	queue := make(chan ManifestFile)
	for w := 0; w < max(workers, 1); w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for f := range queue {
				state, err := fetchOne(base, outDir, f)
				mu.Lock()
				switch {
				case err != nil:
					res.Failed = append(res.Failed, Failure{File: f.Name, Err: err.Error()})
					rich.Error("Giving up on %s: %v.", f.Name, err)
				case state == "kept":
					res.Kept++
				default:
					res.Downloaded++
					rich.Info("Downloaded %s.", f.Name)
				}
				mu.Unlock()
			}
		}()
	}
	for _, f := range m.Files {
		queue <- f
	}
	close(queue)
	wg.Wait()
	if len(res.Failed) == 0 {
		if err := os.WriteFile(filepath.Join(outDir, ManifestName), raw, 0644); err != nil {
			return res, err
		}
	}
	return res, nil
}

func fetchOne(base string, outDir string, f ManifestFile) (string, error) {
	if !validName(f.Name) {
		return "", fmt.Errorf("unexpected file name %q", f.Name)
	}
	want := strings.ToLower(f.Hash)
	dst := filepath.Join(outDir, f.Name)
	if existing, err := os.ReadFile(dst); err == nil && want != "" {
		sum := sha256.Sum256(existing)
		if hex.EncodeToString(sum[:]) == want {
			return "kept", nil
		}
	}
	data, err := network.Bytes(base + "/" + f.Name)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	if want != "" && hex.EncodeToString(sum[:]) != want {
		return "", fmt.Errorf("sha256 differs from the manifest")
	}
	tmp := dst + ".part"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return "", err
	}
	return "downloaded", os.Rename(tmp, dst)
}
