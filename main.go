package main

import (
	"flag"

	"github.com/awadoriproj/awadori-assets/pipeline"
)

func main() {
	var o pipeline.Options
	flag.BoolVar(&o.Update, "update", false, "Download the latest catalog before doing anything else.")
	flag.BoolVar(&o.Keys, "keys", false, "List catalog keys (filtered by -prefix) and exit.")
	flag.StringVar(&o.Get, "get", "", "Only handle the bundles needed by this exact asset key.")
	flag.StringVar(&o.Prefix, "prefix", "", "Only handle assets whose key starts with this prefix.")
	flag.StringVar(&o.FilterRegex, "filter-regex", "", "Only handle bundles whose name or key matches the regex.")
	flag.StringVar(&o.ExcludeRegex, "exclude-regex", "", "Skip bundles whose name or key matches the regex.")
	flag.IntVar(&o.Workers, "workers", pipeline.DefaultWorkers, "Number of parallel downloads.")
	flag.BoolVar(&o.Force, "force", false, "Ignore recorded state and process everything selected again.")
	flag.BoolVar(&o.KeepRaw, "keepraw", false, "Do not delete encrypted raw bundles after decrypting.")
	flag.BoolVar(&o.Convert, "convert", false, "Only decrypt raw bundles already in files/assets without downloading.")
	flag.BoolVar(&o.Extract, "extract", false, "Extract assets from the decrypted bundles into files/extracted.")
	flag.StringVar(&o.Unpack, "unpack", "", "Extract a single Unity archive file, or every file inside a directory, and exit.")
	flag.StringVar(&o.Root, "root", pipeline.DefaultRoot, "CDN root URL.")
	flag.StringVar(&o.Version, "version", pipeline.DefaultVersion, "Catalog version used by -update.")
	flag.StringVar(&o.Encrypt, "encrypt", "", "Encrypt a decrypted bundle file, or every file inside a directory, into files/encrypted and exit.")
	flag.StringVar(&o.ExportServer, "export-server", "", "Write a static tree for the private server (encrypted bundles at their catalog paths plus the catalog) into this directory.")
	flag.StringVar(&o.MasterDownload, "master-download", "", "Download this master data version into -master-out (default files/master/<version>) and exit.")
	flag.StringVar(&o.MasterDecode, "master-decode", "", "Decode the master data .bin files of this directory into JSON tables and exit.")
	flag.StringVar(&o.MasterEncode, "master-encode", "", "Encode the JSON tables of this directory into master data .bin files plus a manifest and exit.")
	flag.StringVar(&o.MasterOut, "master-out", "", "Output directory of the master data commands.")
	flag.StringVar(&o.MasterKey, "master-key", "", "Master data Rijndael-256 key as 64 hex digits (or env AWADORI_MASTER_KEY).")
	flag.StringVar(&o.MasterIV, "master-iv", "", "Master data Rijndael-256 CBC IV as 64 hex digits (or env AWADORI_MASTER_IV).")
	flag.StringVar(&o.MasterVersion, "master-version", "", "Version written to MasterManifest.json by -master-encode.")
	flag.StringVar(&o.MasterPrefix, "master-prefix", "", "Directory of original .bin files whose 64 byte prefixes are reused by -master-encode.")
	flag.StringVar(&o.MasterRoot, "master-root", "", "CDN root URL for -master-download (default: -root).")
	flag.Parse()
	pipeline.Run(o)
}
