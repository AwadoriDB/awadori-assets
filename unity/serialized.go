package unity

import (
	"encoding/binary"
	"fmt"
)

const ClassTextAsset = 49

type Object struct {
	PathID  int64
	Offset  int64
	Size    uint32
	ClassID int32
}

type File struct {
	Objects []Object
	Order   binary.ByteOrder
	data    []byte
}

func ParseSerialized(data []byte) (file *File, err error) {
	defer func() {
		if v := recover(); v != nil {
			file, err = nil, fmt.Errorf("invalid serialized file: %v", v)
		}
	}()
	c := &cursor{b: data, order: binary.BigEndian}
	c.u32()
	c.u32()
	version := c.u32()
	dataOffset := int64(c.u32())
	if version < 13 || version > 30 {
		return nil, fmt.Errorf("unsupported serialized file version %d", version)
	}
	little := true
	endian := c.u8()
	c.skip(3)
	if version >= 22 {
		c.u32()
		c.i64()
		dataOffset = c.i64()
		c.i64()
	}
	if endian == 1 {
		little = false
	}
	if little {
		c.order = binary.LittleEndian
	}
	c.cstr()
	c.skip(4)
	typeTree := c.u8() != 0
	classes := make([]int32, c.i32())
	for i := range classes {
		classID := c.i32()
		classes[i] = classID
		if version >= 16 {
			c.skip(1)
		}
		if version >= 17 {
			c.skip(2)
		}
		if (version < 16 && classID < 0) || (version >= 16 && classID == 114) {
			c.skip(16)
		}
		c.skip(16)
		if typeTree {
			nodes := int(c.i32())
			strings := int(c.i32())
			nodeSize := 24
			if version >= 19 {
				nodeSize = 32
			}
			c.skip(nodes*nodeSize + strings)
			if version >= 21 {
				c.skip(int(c.i32()) * 4)
			}
		}
	}
	bigID := false
	if version < 14 {
		bigID = c.i32() != 0
	}
	objects := make([]Object, c.i32())
	for i := range objects {
		var o Object
		switch {
		case bigID:
			o.PathID = c.i64()
		case version < 14:
			o.PathID = int64(c.i32())
		default:
			c.align(4)
			o.PathID = c.i64()
		}
		if version >= 22 {
			o.Offset = c.i64()
		} else {
			o.Offset = int64(c.u32())
		}
		o.Offset += dataOffset
		o.Size = c.u32()
		typeID := c.i32()
		if version < 16 {
			o.ClassID = int32(c.u16())
		} else {
			if typeID < 0 || int(typeID) >= len(classes) {
				return nil, fmt.Errorf("object type index out of range")
			}
			o.ClassID = classes[typeID]
		}
		if version < 17 {
			c.skip(2)
		}
		if version >= 15 && version < 17 {
			c.skip(1)
		}
		if o.Offset < 0 || o.Offset+int64(o.Size) > int64(len(data)) {
			return nil, fmt.Errorf("object %d out of range", o.PathID)
		}
		objects[i] = o
	}
	return &File{Objects: objects, Order: c.order, data: data}, nil
}

func (f *File) ObjectData(o Object) []byte {
	return f.data[o.Offset : o.Offset+int64(o.Size)]
}

func (f *File) TextAsset(o Object) (name string, script []byte, err error) {
	defer func() {
		if v := recover(); v != nil {
			name, script, err = "", nil, fmt.Errorf("invalid text asset: %v", v)
		}
	}()
	c := &cursor{b: f.ObjectData(o), order: f.Order}
	name = string(c.take(int(c.i32())))
	c.align(4)
	script = c.take(int(c.i32()))
	return name, script, nil
}
