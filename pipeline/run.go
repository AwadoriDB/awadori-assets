package pipeline

import (
	"os"

	"awano-winter/utils"
)

func Run(o Options) {
	for _, d := range []string{assetsDir, decryptedDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			panic(err)
		}
	}
	if o.Unpack != "" {
		UnpackPath(o.Unpack)
		return
	}
	if o.Update {
		updateCatalog(o.Root, o.Version)
	}
	cat := loadCatalog()
	if o.Keys {
		listKeys(cat, o.Prefix)
		return
	}
	bundles := selectBundles(cat, o)
	utils.WriteToJsonFile(bundles, catalogJsonFile)
	transfer(o, bundles)
	if o.Extract {
		extractBundles(bundles)
	}
}
