package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
)

const (
	Window   = 16384
	Provider = "Fwk.Crypt.AssetBundleCryptProvider"
)

var (
	Magic        = []byte("UnityFS\x00")
	ErrNotBundle = errors.New("not a UnityFS bundle after decryption")
	aesKey, _    = hex.DecodeString("7372a4ee777db361ad896c99e408a182")
	nonceSalt, _ = hex.DecodeString("ee24a70238e2a0e5")
)

func keystream(name string, n int) []byte {
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		panic(err)
	}
	sum := sha256.Sum256(append(append([]byte{}, nonceSalt...), name...))
	blocks := (n + 15) / 16
	out := make([]byte, blocks*16)
	var in [16]byte
	copy(in[:8], sum[:8])
	for i := 0; i < blocks; i++ {
		binary.BigEndian.PutUint64(in[8:], uint64(i))
		block.Encrypt(out[i*16:], in[:])
	}
	return out[:n]
}

func Decrypt(data []byte, name string) {
	n := min(len(data), Window)
	mask := keystream(name, n)
	for i := 0; i < n; i++ {
		data[i] ^= mask[i]
	}
}

func DecryptFile(src string, dst string, name string, provider string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	head := make([]byte, Window)
	n, err := io.ReadFull(in, head)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return err
	}
	head = head[:n]
	if provider == Provider && !bytes.HasPrefix(head, Magic) {
		Decrypt(head, name)
	}
	if !bytes.HasPrefix(head, Magic) {
		return ErrNotBundle
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := out.Write(head); err != nil {
		out.Close()
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
