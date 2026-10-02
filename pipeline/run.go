package pipeline

import (
	"os"

	"github.com/awadoriproj/awadori-assets/utils"
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
	if o.Encrypt != "" {
		EncryptPath(o.Encrypt)
		return
	}
	if runMaster(o) {
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
	if o.ExportServer != "" {
		exportServer(o, bundles)
	}
}
