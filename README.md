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

## Layout

```
files/catalog.bin        addressables catalog
files/catalog.json       bundles selected by the last run
files/catalog_diff.json  bundles processed by the last run
files/state.json         completed bundles
files/assets/            raw downloaded bundles (removed after decrypting unless -keepraw)
files/decrypted/         decrypted UnityFS bundles
files/extracted/         extracted assets
```

## Extraction

`unity/` reads UnityFS archives (stored, LZ4, LZ4HC, LZMA) and serialized files natively. `TextAsset` objects are written out as-is (`.bytes` when the name has no extension). Other object types are counted and reported per bundle by class ID.

## License

AGPL-3.0 license

## Credits
### Reference 
[inspix-hailstorm](https://github.com/vertesan/inspix-hailstorm). (ported for awadori)
[moenotes](https://github.com/StarMoe-org/moenotes) (huge reference)
