package catalog

import (
	"encoding/binary"
	"fmt"
	"sort"
	"strings"
)

const AssetMarker = "/asset/Android/"

type Options struct {
	Size uint32
	Crc  uint32
}

type Location struct {
	ID       uint32
	Internal string
	Provider string
	Type     string
	Deps     []uint32
	Opts     *Options
}

type Catalog struct {
	Keys map[string][]uint32
	Locs map[uint32]*Location
}

type Bundle struct {
	Name     string
	Path     string
	Provider string
	Size     uint32
	Crc      uint32
	Keys     []string
}

func Parse(data []byte) (cat *Catalog, err error) {
	defer recoverTo(&err)
	if len(data) < 12 || binary.LittleEndian.Uint32(data) != 0x0de38942 || binary.LittleEndian.Uint32(data[4:]) != 2 {
		return nil, fmt.Errorf("unsupported catalog format")
	}
	r := newReader(data)
	cat = &Catalog{Keys: map[string][]uint32{}, Locs: map[uint32]*Location{}}
	pending := map[uint32]bool{}
	for _, at := range r.array(r.u32(8), 8) {
		var ids []uint32
		for _, x := range r.array(r.u32(at+4), 4) {
			ids = append(ids, r.u32(x))
		}
		for _, id := range ids {
			pending[id] = true
		}
		if k, ok := r.key(r.u32(at)); ok {
			cat.Keys[k] = ids
		}
	}
	for len(pending) > 0 {
		var i uint32
		for i = range pending {
			break
		}
		delete(pending, i)
		if _, ok := cat.Locs[i]; ok {
			continue
		}
		loc := r.location(i)
		cat.Locs[i] = loc
		for _, d := range loc.Deps {
			if _, ok := cat.Locs[d]; !ok {
				pending[d] = true
			}
		}
	}
	return cat, nil
}

func BundleName(l *Location) string {
	return l.Internal[strings.LastIndex(l.Internal, "/")+1:]
}

func IsBuiltin(name string) bool {
	n := strings.ToLower(name)
	return strings.Contains(n, "_monoscripts_") || strings.Contains(n, "_unitybuiltinassets_") ||
		strings.HasSuffix(n, "_monoscripts.bundle") || strings.HasSuffix(n, "_unitybuiltinassets.bundle")
}

func Downloadable(l *Location) bool {
	return l.Opts != nil && strings.HasPrefix(l.Internal, "http") &&
		strings.Contains(l.Internal, AssetMarker) && !IsBuiltin(BundleName(l))
}

func (c *Catalog) Closure(key string) ([]*Location, error) {
	ids, ok := c.Keys[key]
	if !ok {
		return nil, fmt.Errorf("key not found: %s", key)
	}
	if len(ids) == 0 {
		return nil, nil
	}
	todo := []uint32{ids[0]}
	seen := map[uint32]bool{}
	var out []*Location
	for len(todo) > 0 {
		i := todo[len(todo)-1]
		todo = todo[:len(todo)-1]
		if seen[i] {
			continue
		}
		seen[i] = true
		l := c.Locs[i]
		todo = append(todo, l.Deps...)
		if Downloadable(l) {
			out = append(out, l)
		}
	}
	sort.Slice(out, func(a, b int) bool { return out[a].ID < out[b].ID })
	return out, nil
}

func (c *Catalog) SortedKeys(prefix string) []string {
	var out []string
	for k := range c.Keys {
		if strings.HasPrefix(k, prefix) {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

func (c *Catalog) Bundles(keys []string) ([]Bundle, error) {
	byName := map[string]*Bundle{}
	var order []string
	for _, k := range keys {
		locs, err := c.Closure(k)
		if err != nil {
			return nil, err
		}
		if len(c.Keys[k]) == 0 {
			continue
		}
		own := Null
		if primary := c.Locs[c.Keys[k][0]]; len(primary.Deps) > 0 {
			own = primary.Deps[0]
		}
		for _, l := range locs {
			name := BundleName(l)
			b, ok := byName[name]
			if !ok {
				b = &Bundle{
					Name:     name,
					Path:     l.Internal[strings.Index(l.Internal, AssetMarker):],
					Provider: l.Provider,
					Size:     l.Opts.Size,
					Crc:      l.Opts.Crc,
				}
				byName[name] = b
				order = append(order, name)
			}
			if l.ID == own {
				b.Keys = append(b.Keys, k)
			}
		}
	}
	sort.Strings(order)
	out := make([]Bundle, 0, len(order))
	for _, n := range order {
		out = append(out, *byName[n])
	}
	return out, nil
}
