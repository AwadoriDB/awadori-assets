package pipeline

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/awadoriproj/awadori-assets/crypto"
	"github.com/awadoriproj/awadori-assets/rich"
)

const encryptedDir = "files/encrypted"

func EncryptPath(path string) {
	info, err := os.Stat(path)
	if err != nil {
		rich.PanicError("Cannot read the encrypt input.", err)
	}
	if !info.IsDir() {
		encryptOne(path, filepath.Join(encryptedDir, filepath.Base(path)))
		return
	}
	count := 0
	err = filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(path, p)
		if err != nil {
			return err
		}
		if encryptOne(p, filepath.Join(encryptedDir, rel)) {
			count++
		}
		return nil
	})
	if err != nil {
		rich.PanicError("Failed to walk the encrypt input.", err)
	}
	rich.Info("%d bundle(s) encrypted into %s.", count, encryptedDir)
}

func encryptOne(src string, dst string) bool {
	if err := crypto.EncryptFile(src, dst, filepath.Base(src), crypto.Provider); err != nil {
		rich.Warning("Skipping %q: %v.", filepath.Base(src), err)
		return false
	}
	rich.Info("%s -> %s", filepath.Base(src), dst)
	return true
}
