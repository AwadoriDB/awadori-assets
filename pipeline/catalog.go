package pipeline

import (
	"fmt"
	"os"
	"strings"

	"awano-winter/catalog"
	"awano-winter/network"
	"awano-winter/rich"
)

func updateCatalog(root string, version string) {
	base := fmt.Sprintf("%s/asset/Android/catalog_%s_en", strings.TrimRight(root, "/"), version)
	data, err := network.Bytes(base + ".bin")
	if err != nil {
		rich.PanicError("Failed to download catalog.", err)
	}
	if err := os.WriteFile(catalogBinFile, data, 0644); err != nil {
		panic(err)
	}
	rich.Info("Catalog saved: %s (%d bytes).", catalogBinFile, len(data))
	if hash, err := network.Bytes(base + ".hash"); err == nil {
		text := strings.TrimSpace(string(hash))
		if err := os.WriteFile(catalogHashFile, []byte(text), 0644); err != nil {
			panic(err)
		}
		rich.Info("Catalog hash: %s.", text)
	}
}

func loadCatalog() *catalog.Catalog {
	data, err := os.ReadFile(catalogBinFile)
	if err != nil {
		rich.Panic("%s is missing, run with -update first.", catalogBinFile)
	}
	cat, err := catalog.Parse(data)
	if err != nil {
		rich.PanicError("Failed to parse catalog.", err)
	}
	return cat
}

func listKeys(cat *catalog.Catalog, prefix string) {
	for _, k := range cat.SortedKeys(prefix) {
		locs, err := cat.Closure(k)
		if err != nil {
			continue
		}
		fmt.Printf("%s\t%d bundle(s)\n", k, len(locs))
	}
}
