package unity

import "encoding/binary"

type cursor struct {
	b     []byte
	pos   int
	order binary.ByteOrder
}

func (c *cursor) take(n int) []byte {
	if n < 0 || c.pos+n > len(c.b) {
		panic("unexpected end of data")
	}
	out := c.b[c.pos : c.pos+n]
	c.pos += n
	return out
}

func (c *cursor) skip(n int) {
	c.take(n)
}

func (c *cursor) align(n int) {
	if r := c.pos % n; r != 0 {
		c.skip(n - r)
	}
}

func (c *cursor) u8() uint8 {
	return c.take(1)[0]
}

func (c *cursor) u16() uint16 {
	return c.order.Uint16(c.take(2))
}

func (c *cursor) u32() uint32 {
	return c.order.Uint32(c.take(4))
}

func (c *cursor) i32() int32 {
	return int32(c.u32())
}

func (c *cursor) i64() int64 {
	return int64(c.order.Uint64(c.take(8)))
}

func (c *cursor) cstr() string {
	start := c.pos
	for c.pos < len(c.b) && c.b[c.pos] != 0 {
		c.pos++
	}
	if c.pos >= len(c.b) {
		panic("unterminated string")
	}
	s := string(c.b[start:c.pos])
	c.pos++
	return s
}
