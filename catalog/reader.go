package catalog

import (
	"encoding/binary"
	"fmt"
	"strings"
	"unicode/utf16"
)

const Null uint32 = 0xFFFFFFFF

type stringKey struct {
	off uint32
	sep string
}

type reader struct {
	d     []byte
	cache map[stringKey]string
}

func newReader(data []byte) *reader {
	return &reader{d: data, cache: map[stringKey]string{}}
}

func (r *reader) u32(o uint32) uint32 {
	return binary.LittleEndian.Uint32(r.d[o : o+4])
}

func (r *reader) array(off uint32, stride uint32) []uint32 {
	if off == Null {
		return nil
	}
	size := r.u32(off - 4)
	if size%stride != 0 || size/stride > 200000 {
		panic("catalog array limit")
	}
	out := make([]uint32, size/stride)
	for i := range out {
		out[i] = off + uint32(i)*stride
	}
	return out
}

func (r *reader) str(off uint32, sep string) string {
	if off == Null {
		return ""
	}
	ck := stringKey{off, sep}
	if v, ok := r.cache[ck]; ok {
		return v
	}
	var v string
	if off&0x40000000 != 0 {
		var parts []string
		seen := map[uint32]bool{}
		p := off
		for p != Null {
			at := p & 0x3fffffff
			if seen[at] || len(seen) > 4096 {
				panic("cyclic string")
			}
			seen[at] = true
			parts = append(parts, r.str(r.u32(at), "\x00"))
			p = r.u32(at + 4)
		}
		for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
			parts[i], parts[j] = parts[j], parts[i]
		}
		v = strings.Join(parts, sep)
	} else {
		at := off & 0x3fffffff
		raw := r.d[at : at+r.u32(at-4)]
		if off&0x80000000 != 0 {
			units := make([]uint16, len(raw)/2)
			for i := range units {
				units[i] = binary.LittleEndian.Uint16(raw[i*2:])
			}
			v = string(utf16.Decode(units))
		} else {
			v = string(raw)
		}
	}
	r.cache[ck] = v
	return v
}

func (r *reader) typ(off uint32) string {
	return r.str(r.u32(off+4), ".")
}

func (r *reader) key(off uint32) (string, bool) {
	kind := r.typ(r.u32(off))
	at := r.u32(off + 4)
	if kind != "System.String" {
		return "", false
	}
	sep := string(rune(binary.LittleEndian.Uint16(r.d[at+4:])))
	return r.str(r.u32(at), sep), true
}

func (r *reader) location(i uint32) *Location {
	loc := &Location{
		ID:       i,
		Internal: r.str(r.u32(i+4), "/"),
		Provider: r.str(r.u32(i+8), "."),
		Type:     r.typ(r.u32(i + 24)),
	}
	for _, x := range r.array(r.u32(i+12), 4) {
		loc.Deps = append(loc.Deps, r.u32(x))
	}
	if extra := r.u32(i + 20); extra != Null {
		at := r.u32(extra + 4)
		loc.Opts = &Options{Size: r.u32(at + 12), Crc: r.u32(at + 8)}
	}
	return loc
}

func recoverTo(err *error) {
	if v := recover(); v != nil {
		*err = fmt.Errorf("catalog parse failed: %v", v)
	}
}
