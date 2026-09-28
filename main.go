package main

import (
	"flag"

	"awano-winter/pipeline"
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
	flag.Parse()
	pipeline.Run(o)
}
