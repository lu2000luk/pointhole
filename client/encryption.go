package main

import (
	"crypto/rand"
	"errors"
	"io"

	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/chacha20poly1305"
)

func Encrypt(plain []byte, pass string) ([]byte, error) {
	salt := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}
	// fast key from password, keyed by salt
	h, _ := blake2b.New256(salt)
	h.Write([]byte(pass))
	key := h.Sum(nil)

	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, chacha20poly1305.NonceSizeX)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	ct := aead.Seal(nil, nonce, plain, nil)

	// blob = salt || nonce || ciphertext
	out := make([]byte, 0, len(salt)+len(nonce)+len(ct))
	out = append(out, salt...)
	out = append(out, nonce...)
	out = append(out, ct...)
	return out, nil
}

func Decrypt(blob []byte, pass string) ([]byte, error) {
	if len(blob) < 16+24+16 { // salt + nonce + tag
		return nil, errors.New("blob too short")
	}
	salt := blob[:16]
	nonce := blob[16:40]
	ct := blob[40:]

	h, _ := blake2b.New256(salt)
	h.Write([]byte(pass))
	key := h.Sum(nil)

	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, err
	}
	plain, err := aead.Open(nil, nonce, ct, nil)
	if err != nil {
		return nil, errors.New("decrypt failed")
	}
	return plain, nil
}
