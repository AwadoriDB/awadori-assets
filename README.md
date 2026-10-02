# awadori

Asset toolkit for BanG Dream! Our Notes, inspired by [inspix-hailstorm](https://github.com/vertesan/inspix-hailstorm).
Reads the Addressables catalog, downloads and decrypts bundles, and extracts assets from Unity archives. Pure Go, no Python.

## Build

```bash
go build -o awadori .
```

## Usage

```bash
./awadori -update
./awadori -keys -prefix Live/MusicScore/
./awadori -get Live/MusicScore/0001/0001_03 -extract
./awadori -prefix Live/MusicScore/ -extract
./awadori -filter-regex "musicscore_00" -extract
./awadori -prefix Live/ -filter-regex "MusicScore|Jacket" -exclude-regex "_en$" -workers 20 -extract
./awadori -unpack some.bundle
./awadori -unpack path/to/dir
```

`-workers` sets how many bundles download in parallel (default 10). `-filter-regex` keeps only bundles whose name or asset key matches, `-exclude-regex` drops the ones that match. They can be combined.

Without `-get`, `-prefix` or `-filter-regex`, every bundle in the catalog is processed. Bundles already downloaded with the same size and CRC are skipped unless `-force` is given.

## Encrypting bundles

The bundle cipher is an XOR of the first 16384 bytes with a keystream derived from the bundle file name, so encrypting is the same operation as decrypting. The file name matters: a bundle must be served under exactly the name the catalog gives it.

```bash
./awadori -encrypt path/to/some.bundle
./awadori -encrypt path/to/decrypted_dir
```

Output goes to `files/encrypted/`. Inputs that do not start with the `UnityFS` magic are skipped, so an already encrypted file is never encrypted twice. In Go, `crypto.EncryptFile` writes a file and `crypto.OpenEncrypted` returns an `io.ReaderAt` that yields the encrypted bytes on the fly, so a server can pass decrypted bundles to `http.ServeContent` without storing a second copy.

## Master data

A master data file is a 64 byte prefix followed by Rijndael-256 (256-bit block, 256-bit key) CBC with PKCS7 padding, wrapping gzip-compressed JSON of the form `{"_allData": [...]}`. The key and IV are not part of this repository; pass them as 64 hex digits each with `-master-key` and `-master-iv`, or through `AWADORI_MASTER_KEY` and `AWADORI_MASTER_IV`.

```bash
./awadori -master-download 1.0.0.105
./awadori -master-decode files/master/1.0.0.105
./awadori -master-encode files/master_json -master-version 1.0.0.105 -master-prefix files/master/1.0.0.105
```

- `-master-download VERSION` fetches `<root>/master/<VERSION>/MasterManifest.json` and every listed `.bin`, checks each SHA-256 and keeps files that are already correct. `-master-root` overrides the CDN root, which defaults to `-root`.
- `-master-decode DIR` writes one indented JSON table per `.bin` into `files/master_json` (or `-master-out`).
- `-master-encode DIR` turns edited JSON tables back into `.bin` files and writes a fresh `MasterManifest.json` into `files/master_bin` (or `-master-out`). `-master-prefix` reuses the 64 byte prefix of the original file of the same name; without it the prefix is zero filled.

## Serving from a private server

```bash
./awadori -prefix Live/ -export-server files/served
```

This downloads and decrypts the selected bundles as usual, then writes a static tree under `files/served`: every bundle re-encrypted at its catalog path (`asset/Android/...`) and the catalog as `asset/Android/catalog_<version>_en.bin` and `.hash`. Serve that directory (and the output of `-master-encode` under `master/<version>/`) from the host the client is redirected to.

The client also asks the game API for the current master data version and resource version before it downloads anything, so the private server has to answer that call with the version you serve.

## Layout

```
files/catalog.bin        addressables catalog
files/catalog.json       bundles selected by the last run
files/catalog_diff.json  bundles processed by the last run
files/state.json         completed bundles
files/assets/            raw downloaded bundles (removed after decrypting unless -keepraw)
files/decrypted/         decrypted UnityFS bundles
files/extracted/         extracted assets
files/encrypted/         output of -encrypt
files/master*/           master data downloads, JSON tables and re-encoded .bin files
files/served/            suggested -export-server target
```

## Extraction

`unity/` reads UnityFS archives (stored, LZ4, LZ4HC, LZMA) and serialized files natively. `TextAsset` objects are written out as-is (`.bytes` when the name has no extension). Other object types are counted and reported per bundle by class ID.

## License

AGPL-3.0 license

## Credits
### Reference 
[inspix-hailstorm](https://github.com/vertesan/inspix-hailstorm). (ported for awadori)
[moenotes](https://github.com/StarMoe-org/moenotes) (huge reference)
