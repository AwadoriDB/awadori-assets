package unity

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/ulikunitz/xz/lzma"
)

var archiveSignature = []byte("UnityFS\x00")

type Node struct {
	Offset int64
	Size   int64
	Flags  uint32
	Path   string
}

type Archive struct {
	Nodes []Node
	data  []byte
}

func IsArchive(head []byte) bool {
	return bytes.HasPrefix(head, archiveSignature)
}

func (a *Archive) NodeData(n Node) []byte {
	return a.data[n.Offset : n.Offset+n.Size]
}

func decompress(mode uint32, raw []byte, size int) ([]byte, error) {
	switch mode {
	case 0:
		if len(raw) != size {
			return nil, fmt.Errorf("stored block size mismatch")
		}
		return raw, nil
	case 1:
		if len(raw) < 5 {
			return nil, fmt.Errorf("truncated lzma block")
		}
		header := make([]byte, 13)
		copy(header, raw[:5])
		binary.LittleEndian.PutUint64(header[5:], uint64(size))
		r, err := lzma.NewReader(io.MultiReader(bytes.NewReader(header), bytes.NewReader(raw[5:])))
		if err != nil {
			return nil, err
		}
		out := make([]byte, size)
		if _, err := io.ReadFull(r, out); err != nil {
			return nil, err
		}
		return out, nil
	case 2, 3:
		return lz4Decode(raw, size)
	}
	return nil, fmt.Errorf("unsupported compression mode %d", mode)
}

func OpenArchive(data []byte) (arc *Archive, err error) {
	defer func() {
		if v := recover(); v != nil {
			arc, err = nil, fmt.Errorf("invalid archive: %v", v)
		}
	}()
	c := &cursor{b: data, order: binary.BigEndian}
	if c.cstr() != "UnityFS" {
		return nil, fmt.Errorf("not a UnityFS archive")
	}
	version := c.u32()
	c.cstr()
	c.cstr()
	c.i64()
	compressedSize := int(c.u32())
	uncompressedSize := int(c.u32())
	flags := c.u32()
	if flags&0x100 != 0 {
		return nil, fmt.Errorf("archive uses Unity's own encryption")
	}
	if version >= 7 {
		c.align(16)
	}
	var infoRaw []byte
	if flags&0x80 != 0 {
		if compressedSize > len(data) {
			return nil, fmt.Errorf("block info larger than file")
		}
		infoRaw = data[len(data)-compressedSize:]
	} else {
		infoRaw = c.take(compressedSize)
	}
	if flags&0x200 != 0 {
		c.align(16)
	}
	info, err := decompress(flags&0x3f, infoRaw, uncompressedSize)
	if err != nil {
		return nil, fmt.Errorf("block info: %w", err)
	}
	ic := &cursor{b: info, order: binary.BigEndian}
	ic.skip(16)
	type block struct {
		size, compressed uint32
		flags            uint16
	}
	blocks := make([]block, ic.i32())
	for i := range blocks {
		blocks[i] = block{ic.u32(), ic.u32(), ic.u16()}
	}
	nodes := make([]Node, ic.i32())
	for i := range nodes {
		nodes[i] = Node{Offset: ic.i64(), Size: ic.i64(), Flags: ic.u32(), Path: ic.cstr()}
	}
	var out []byte
	for _, b := range blocks {
		part, err := decompress(uint32(b.flags)&0x3f, c.take(int(b.compressed)), int(b.size))
		if err != nil {
			return nil, err
		}
		out = append(out, part...)
	}
	for _, n := range nodes {
		if n.Offset < 0 || n.Size < 0 || n.Offset+n.Size > int64(len(out)) {
			return nil, fmt.Errorf("node %q out of range", n.Path)
		}
	}
	return &Archive{Nodes: nodes, data: out}, nil
}
