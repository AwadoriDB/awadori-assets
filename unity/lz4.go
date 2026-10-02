package unity

import "errors"

var errLz4 = errors.New("corrupt lz4 block")

func lz4Decode(src []byte, size int) ([]byte, error) {
	dst := make([]byte, 0, size)
	i := 0
	extend := func(n int) (int, bool) {
		for {
			if i >= len(src) {
				return 0, false
			}
			b := int(src[i])
			i++
			n += b
			if b != 255 {
				return n, true
			}
		}
	}
	for i < len(src) {
		token := int(src[i])
		i++
		lit := token >> 4
		if lit == 15 {
			var ok bool
			if lit, ok = extend(lit); !ok {
				return nil, errLz4
			}
		}
		if i+lit > len(src) {
			return nil, errLz4
		}
		dst = append(dst, src[i:i+lit]...)
		i += lit
		if i >= len(src) {
			break
		}
		if i+2 > len(src) {
			return nil, errLz4
		}
		offset := int(src[i]) | int(src[i+1])<<8
		i += 2
		if offset == 0 || offset > len(dst) {
			return nil, errLz4
		}
		match := token & 15
		if match == 15 {
			var ok bool
			if match, ok = extend(match); !ok {
				return nil, errLz4
			}
		}
		match += 4
		from := len(dst) - offset
		for k := 0; k < match; k++ {
			dst = append(dst, dst[from+k])
		}
	}
	if len(dst) != size {
		return nil, errLz4
	}
	return dst, nil
}
