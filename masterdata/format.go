package masterdata

import (
	"bytes"
	"compress/gzip"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

const PrefixSize = 64

var (
	ErrShortFile  = errors.New("file is shorter than the 64 byte prefix")
	ErrBadPadding = errors.New("bad PKCS7 padding, wrong key or IV or not a master data file")
	ErrBadJSON    = errors.New("table is not valid JSON")
)

type Key struct {
	Cipher *Cipher
	IV     []byte
}

func NewKey(keyHex string, ivHex string) (*Key, error) {
	key, err := hex.DecodeString(keyHex)
	if err != nil || len(key) != KeySize {
		return nil, fmt.Errorf("master key must be %d bytes as hex", KeySize)
	}
	iv, err := hex.DecodeString(ivHex)
	if err != nil || len(iv) != BlockSize {
		return nil, fmt.Errorf("master IV must be %d bytes as hex", BlockSize)
	}
	c, err := NewCipher(key)
	if err != nil {
		return nil, err
	}
	return &Key{Cipher: c, IV: iv}, nil
}

func Decode(raw []byte, k *Key) ([]byte, error) {
	if len(raw) <= PrefixSize {
		return nil, ErrShortFile
	}
	plain, err := k.Cipher.DecryptCBC(raw[PrefixSize:], k.IV)
	if err != nil {
		return nil, err
	}
	pad := int(plain[len(plain)-1])
	if pad < 1 || pad > BlockSize || pad > len(plain) {
		return nil, ErrBadPadding
	}
	if !bytes.Equal(plain[len(plain)-pad:], bytes.Repeat([]byte{byte(pad)}, pad)) {
		return nil, ErrBadPadding
	}
	zr, err := gzip.NewReader(bytes.NewReader(plain[:len(plain)-pad]))
	if err != nil {
		return nil, fmt.Errorf("gzip: %w", err)
	}
	defer zr.Close()
	return io.ReadAll(zr)
}

func Encode(jsonData []byte, k *Key, prefix []byte) ([]byte, error) {
	if !json.Valid(jsonData) {
		return nil, ErrBadJSON
	}
	var zbuf bytes.Buffer
	zw := gzip.NewWriter(&zbuf)
	if _, err := zw.Write(jsonData); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	pad := BlockSize - zbuf.Len()%BlockSize
	zbuf.Write(bytes.Repeat([]byte{byte(pad)}, pad))
	enc, err := k.Cipher.EncryptCBC(zbuf.Bytes(), k.IV)
	if err != nil {
		return nil, err
	}
	head := make([]byte, PrefixSize)
	copy(head, prefix)
	return append(head, enc...), nil
}

func Pretty(data []byte) ([]byte, error) {
	var out bytes.Buffer
	if err := json.Indent(&out, data, "", "  "); err != nil {
		return nil, err
	}
	out.WriteByte('\n')
	return out.Bytes(), nil
}

func Compact(data []byte) ([]byte, error) {
	var out bytes.Buffer
	if err := json.Compact(&out, data); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
