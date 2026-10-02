package pipeline

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/awadoriproj/awadori-assets/catalog"
	"github.com/awadoriproj/awadori-assets/crypto"
	"github.com/awadoriproj/awadori-assets/rich"
)

func copyIfExists(src string, dst string) bool {
	data, err := os.ReadFile(src)
	if err != nil {
		return false
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		panic(err)
	}
	if err := os.WriteFile(dst, data, 0644); err != nil {
		panic(err)
	}
	return true
}

func exportServer(o Options, bundles []catalog.Bundle) {
	root := o.ExportServer
	count, missing := 0, 0
	for _, b := range bundles {
		src := filepath.Join(decryptedDir, b.Name)
		if _, err := os.Stat(src); err != nil {
			missing++
			continue
		}
		dst := filepath.Join(root, filepath.FromSlash(b.Path))
		if err := crypto.EncryptFile(src, dst, b.Name, b.Provider); err != nil {
			rich.Warning("Skipping %q: %v.", b.Name, err)
			continue
		}
		count++
	}
	base := filepath.Join(root, "asset", "Android", fmt.Sprintf("catalog_%s_en", o.Version))
	copyIfExists(catalogBinFile, base+".bin")
	copyIfExists(catalogHashFile, base+".hash")
	if missing > 0 {
		rich.Warning("%d selected bundle(s) are not in %s and were not exported.", missing, decryptedDir)
	}
	rich.Info("%d bundle(s) and the catalog exported into %s.", count, root)
}
