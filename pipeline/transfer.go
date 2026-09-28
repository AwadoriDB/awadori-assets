package pipeline

import (
	"os"
	"path/filepath"
	"strings"

	"awano-winter/catalog"
	"awano-winter/crypto"
	"awano-winter/network"
	"awano-winter/rich"
	"awano-winter/utils"
)

type doneState struct {
	Size uint32
	Crc  uint32
}

func loadState() map[string]doneState {
	state := map[string]doneState{}
	if err := utils.ReadFromJsonFile(stateFile, &state); err != nil && !os.IsNotExist(err) {
		panic(err)
	}
	return state
}

func pendingBundles(bundles []catalog.Bundle, state map[string]doneState, force bool) []catalog.Bundle {
	var out []catalog.Bundle
	for _, b := range bundles {
		if !force {
			if s, ok := state[b.Name]; ok && s.Size == b.Size && s.Crc == b.Crc {
				if _, err := os.Stat(filepath.Join(decryptedDir, b.Name)); err == nil {
					continue
				}
			}
		}
		out = append(out, b)
	}
	return out
}

func download(pending []catalog.Bundle, root string, workers int) []catalog.Bundle {
	root = strings.TrimRight(root, "/")
	jobs := make([]network.Job, len(pending))
	for i, b := range pending {
		jobs[i] = network.Job{
			Name: b.Name,
			URL:  root + b.Path,
			Dst:  filepath.Join(assetsDir, b.Name),
			Size: int64(b.Size),
		}
	}
	failed := map[int]bool{}
	for _, i := range network.Download(jobs, max(workers, 1), func(int) {}) {
		failed[i] = true
	}
	var ready []catalog.Bundle
	for i, b := range pending {
		if !failed[i] {
			ready = append(ready, b)
		}
	}
	if len(failed) > 0 {
		rich.Error("%d bundle(s) failed to download.", len(failed))
	}
	return ready
}

func existingRaw(pending []catalog.Bundle) []catalog.Bundle {
	var ready []catalog.Bundle
	for _, b := range pending {
		if _, err := os.Stat(filepath.Join(assetsDir, b.Name)); err == nil {
			ready = append(ready, b)
		}
	}
	return ready
}

func decryptAll(ready []catalog.Bundle, state map[string]doneState, keepRaw bool) []catalog.Bundle {
	var processed []catalog.Bundle
	for i, b := range ready {
		src := filepath.Join(assetsDir, b.Name)
		dst := filepath.Join(decryptedDir, b.Name)
		if err := crypto.DecryptFile(src, dst, b.Name, b.Provider); err != nil {
			rich.Warning("Skipping %q: %v.", b.Name, err)
			continue
		}
		state[b.Name] = doneState{Size: b.Size, Crc: b.Crc}
		processed = append(processed, b)
		if !keepRaw {
			os.Remove(src)
		}
		if (i+1)%500 == 0 || i+1 == len(ready) {
			rich.Info("(%d/%d) Decrypted.", i+1, len(ready))
		}
	}
	return processed
}

func transfer(o Options, bundles []catalog.Bundle) {
	state := loadState()
	pending := pendingBundles(bundles, state, o.Force)
	if len(pending) == 0 {
		rich.Info("Nothing is updated, will be stopping process.")
		return
	}
	rich.Info("%d bundle(s) to process.", len(pending))
	var ready []catalog.Bundle
	if o.Convert {
		ready = existingRaw(pending)
	} else {
		ready = download(pending, o.Root, o.Workers)
	}
	processed := decryptAll(ready, state, o.KeepRaw)
	utils.WriteToJsonFile(state, stateFile)
	utils.WriteToJsonFile(processed, catalogDiffFile)
	if err := os.WriteFile(updatedFlagFile, nil, 0644); err != nil {
		panic(err)
	}
	rich.Info("All bundles decrypted into %s.", decryptedDir)
}
