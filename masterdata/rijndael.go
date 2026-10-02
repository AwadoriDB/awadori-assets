package masterdata

import (
	"errors"
	"math/bits"
)

const (
	BlockSize = 32
	KeySize   = 32
	nb        = 8
	nk        = 8
	nr        = 14
)

var shifts = [4]int{0, 1, 3, 4}

var (
	sbox, invSbox             [256]byte
	mul2, mul3                [256]byte
	mul9, mul11, mul13, mul14 [256]byte
)

func xtime(a byte) byte {
	if a&0x80 != 0 {
		return a<<1 ^ 0x1b
	}
	return a << 1
}

func gmul(a, b byte) byte {
	var r byte
	for b != 0 {
		if b&1 != 0 {
			r ^= a
		}
		a = xtime(a)
		b >>= 1
	}
	return r
}

func init() {
	var p, q byte

	p, q = 1, 1
	for {
		hi := byte(0)
		if p&0x80 != 0 {
			hi = 0x1b
		}
		p = p ^ p<<1 ^ hi
		q ^= q << 1
		q ^= q << 2
		q ^= q << 4
		if q&0x80 != 0 {
			q ^= 0x09
		}
		x := q ^ bits.RotateLeft8(q, 1) ^ bits.RotateLeft8(q, 2) ^ bits.RotateLeft8(q, 3) ^ bits.RotateLeft8(q, 4)
		sbox[p] = x ^ 0x63
		if p == 1 {
			break
		}
	}
	sbox[0] = 0x63
	for i := 0; i < 256; i++ {
		invSbox[sbox[i]] = byte(i)
		b := byte(i)
		mul2[i], mul3[i] = gmul(b, 2), gmul(b, 3)
		mul9[i], mul11[i], mul13[i], mul14[i] = gmul(b, 9), gmul(b, 11), gmul(b, 13), gmul(b, 14)
	}
}

type Cipher struct {
	rk [nr + 1][4][nb]byte
}

func NewCipher(key []byte) (*Cipher, error) {
	if len(key) != KeySize {
		return nil, errors.New("masterdata: key must be 32 bytes")
	}
	words := make([][4]byte, nb*(nr+1))
	for i := 0; i < nk; i++ {
		copy(words[i][:], key[4*i:4*i+4])
	}
	rcon := byte(1)
	for i := nk; i < len(words); i++ {
		t := words[i-1]
		switch {
		case i%nk == 0:
			t = [4]byte{t[1], t[2], t[3], t[0]}
			for j := range t {
				t[j] = sbox[t[j]]
			}
			t[0] ^= rcon
			rcon = xtime(rcon)
		case i%nk == 4:
			for j := range t {
				t[j] = sbox[t[j]]
			}
		}
		for j := 0; j < 4; j++ {
			words[i][j] = words[i-nk][j] ^ t[j]
		}
	}
	c := &Cipher{}
	for rnd := 0; rnd <= nr; rnd++ {
		for col := 0; col < nb; col++ {
			for row := 0; row < 4; row++ {
				c.rk[rnd][row][col] = words[rnd*nb+col][row]
			}
		}
	}
	return c, nil
}

type state [4][nb]byte

func load(b []byte) (s state) {
	for c := 0; c < nb; c++ {
		for r := 0; r < 4; r++ {
			s[r][c] = b[4*c+r]
		}
	}
	return
}

func (s *state) store(b []byte) {
	for c := 0; c < nb; c++ {
		for r := 0; r < 4; r++ {
			b[4*c+r] = s[r][c]
		}
	}
}

func (s *state) addKey(k *[4][nb]byte) {
	for r := 0; r < 4; r++ {
		for c := 0; c < nb; c++ {
			s[r][c] ^= k[r][c]
		}
	}
}

func (s *state) shiftRows(inv bool) {
	var t [nb]byte
	for r := 1; r < 4; r++ {
		for c := 0; c < nb; c++ {
			if inv {
				t[c] = s[r][(c-shifts[r]+nb)%nb]
			} else {
				t[c] = s[r][(c+shifts[r])%nb]
			}
		}
		s[r] = t
	}
}

func (s *state) sub(box *[256]byte) {
	for r := 0; r < 4; r++ {
		for c := 0; c < nb; c++ {
			s[r][c] = box[s[r][c]]
		}
	}
}

func (s *state) mix() {
	for c := 0; c < nb; c++ {
		a0, a1, a2, a3 := s[0][c], s[1][c], s[2][c], s[3][c]
		s[0][c] = mul2[a0] ^ mul3[a1] ^ a2 ^ a3
		s[1][c] = a0 ^ mul2[a1] ^ mul3[a2] ^ a3
		s[2][c] = a0 ^ a1 ^ mul2[a2] ^ mul3[a3]
		s[3][c] = mul3[a0] ^ a1 ^ a2 ^ mul2[a3]
	}
}

func (s *state) invMix() {
	for c := 0; c < nb; c++ {
		a0, a1, a2, a3 := s[0][c], s[1][c], s[2][c], s[3][c]
		s[0][c] = mul14[a0] ^ mul11[a1] ^ mul13[a2] ^ mul9[a3]
		s[1][c] = mul9[a0] ^ mul14[a1] ^ mul11[a2] ^ mul13[a3]
		s[2][c] = mul13[a0] ^ mul9[a1] ^ mul14[a2] ^ mul11[a3]
		s[3][c] = mul11[a0] ^ mul13[a1] ^ mul9[a2] ^ mul14[a3]
	}
}

func (c *Cipher) EncryptBlock(dst, src []byte) {
	s := load(src)
	s.addKey(&c.rk[0])
	for rnd := 1; rnd <= nr; rnd++ {
		s.sub(&sbox)
		s.shiftRows(false)
		if rnd < nr {
			s.mix()
		}
		s.addKey(&c.rk[rnd])
	}
	s.store(dst)
}

func (c *Cipher) DecryptBlock(dst, src []byte) {
	s := load(src)
	s.addKey(&c.rk[nr])
	for rnd := nr - 1; rnd >= 0; rnd-- {
		s.shiftRows(true)
		s.sub(&invSbox)
		s.addKey(&c.rk[rnd])
		if rnd > 0 {
			s.invMix()
		}
	}
	s.store(dst)
}

func (c *Cipher) EncryptCBC(data, iv []byte) ([]byte, error) {
	if len(iv) != BlockSize || len(data)%BlockSize != 0 {
		return nil, errors.New("masterdata: iv must be 32 bytes and data a multiple of 32")
	}
	out := make([]byte, len(data))
	prev := append([]byte{}, iv...)
	var x [BlockSize]byte
	for i := 0; i < len(data); i += BlockSize {
		for j := 0; j < BlockSize; j++ {
			x[j] = data[i+j] ^ prev[j]
		}
		c.EncryptBlock(out[i:i+BlockSize], x[:])
		prev = out[i : i+BlockSize]
	}
	return out, nil
}

func (c *Cipher) DecryptCBC(data, iv []byte) ([]byte, error) {
	if len(iv) != BlockSize || len(data) == 0 || len(data)%BlockSize != 0 {
		return nil, errors.New("masterdata: iv must be 32 bytes and ciphertext a positive multiple of 32")
	}
	out := make([]byte, len(data))
	prev := iv
	for i := 0; i < len(data); i += BlockSize {
		c.DecryptBlock(out[i:i+BlockSize], data[i:i+BlockSize])
		for j := 0; j < BlockSize; j++ {
			out[i+j] ^= prev[j]
		}
		prev = data[i : i+BlockSize]
	}
	return out, nil
}
