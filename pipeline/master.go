package pipeline

import (
	"os"
	"path/filepath"

	"github.com/awadoriproj/awadori-assets/masterdata"
	"github.com/awadoriproj/awadori-assets/rich"
)

const (
	masterDir     = "files/master"
	masterJSONDir = "files/master_json"
	masterBinDir  = "files/master_bin"
)

func masterKey(o Options) *masterdata.Key {
	key, iv := o.MasterKey, o.MasterIV
	if key == "" {
		key = os.Getenv("AWADORI_MASTER_KEY")
	}
	if iv == "" {
		iv = os.Getenv("AWADORI_MASTER_IV")
	}
	k, err := masterdata.NewKey(key, iv)
	if err != nil {
		rich.PanicError("Master data key is missing or invalid, use -master-key and -master-iv.", err)
	}
	return k
}

func reportMaster(verb string, res *masterdata.Result, out string) {
	for _, f := range res.Failed {
		rich.Error("%s failed: %s.", f.File, f.Err)
	}
	rich.Info("%d table(s) %s into %s, %d failed.", res.Done, verb, out, len(res.Failed))
}

func runMaster(o Options) bool {
	switch {
	case o.MasterDownload != "":
		out := o.MasterOut
		if out == "" {
			out = filepath.Join(masterDir, o.MasterDownload)
		}
		root := o.MasterRoot
		if root == "" {
			root = o.Root
		}
		res, err := masterdata.Download(root, o.MasterDownload, out, o.Workers)
		if err != nil {
			rich.PanicError("Master data download failed.", err)
		}
		rich.Info("Master data %s: %d downloaded, %d kept, %d failed.", o.MasterDownload, res.Downloaded, res.Kept, len(res.Failed))
	case o.MasterDecode != "":
		out := o.MasterOut
		if out == "" {
			out = masterJSONDir
		}
		res, err := masterdata.DecodeDir(o.MasterDecode, out, masterKey(o), true, o.Workers)
		if err != nil {
			rich.PanicError("Master data decode failed.", err)
		}
		reportMaster("decoded", res, out)
	case o.MasterEncode != "":
		out := o.MasterOut
		if out == "" {
			out = masterBinDir
		}
		res, err := masterdata.EncodeDir(o.MasterEncode, out, masterKey(o), o.MasterPrefix, o.Workers)
		if err != nil {
			rich.PanicError("Master data encode failed.", err)
		}
		reportMaster("encoded", res, out)
		if len(res.Failed) > 0 {
			return true
		}
		version := o.MasterVersion
		if version == "" {
			rich.Warning("No -master-version given, the manifest version is left empty.")
		}
		m, err := masterdata.BuildManifest(out, version)
		if err != nil {
			rich.PanicError("Failed to build the manifest.", err)
		}
		if err := masterdata.WriteManifest(out, m); err != nil {
			rich.PanicError("Failed to write the manifest.", err)
		}
		rich.Info("%s written with %d file(s).", masterdata.ManifestName, len(m.Files))
	default:
		return false
	}
	return true
}
