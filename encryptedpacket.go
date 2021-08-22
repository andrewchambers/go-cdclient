package collectd

import (
	"bufio"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"io"
)

type EncryptedPacket struct {
	PlainTextPacket
	username         []byte
	ebuffer          bytes.Buffer
	aesBlockCipher   cipher.Block
	iv               []byte
	cryptoRandReader *bufio.Reader
}

func NewEncryptedPacket(username, password string) (*EncryptedPacket, error) {
	return NewEncryptedPacketSize(username, password, DefaultBufferSize)
}

func NewEncryptedPacketSize(username, password string, size int) (*EncryptedPacket, error) {
	if len(username) > 64 {
		return nil, errors.New("username must be 0-64 characters")
	}
	if len(password) > 64 {
		return nil, errors.New("password must be 0-64 characters")
	}
	b := &EncryptedPacket{}
	b.username = []byte(username)
	passwordHash := sha256.Sum256([]byte(password))
	aesBlockCipher, err := aes.NewCipher(passwordHash[:])
	if err != nil {
		return nil, err
	}
	// Batching up crypto/rand reads makes then far faster.
	// See https://github.com/golang/go/issues/16593
	b.cryptoRandReader = bufio.NewReader(rand.Reader)
	b.aesBlockCipher = aesBlockCipher
	b.iv = make([]byte, 16)
	b.PlainTextPacket.init(size - 42 - len(username))
	return b, nil
}

func (b *EncryptedPacket) Finalize() []byte {
	if b.buffer.Len() == 0 {
		return nil
	}
	plainText := b.buffer.Bytes()
	b.ebuffer.Reset()
	if _, err := io.ReadFull(b.cryptoRandReader, b.iv); err != nil {
		panic(err) // just die if this ever happens.
	}
	size := uint16(42 + len(b.username) + len(plainText))
	tmp := [6]byte{}
	binary.BigEndian.PutUint16(tmp[0:2], uint16(typeEncryptAES256))
	binary.BigEndian.PutUint16(tmp[2:4], size)
	binary.BigEndian.PutUint16(tmp[4:6], uint16(len(b.username)))
	b.ebuffer.Write(tmp[:])
	b.ebuffer.Write(b.username)
	b.ebuffer.Write(b.iv)
	pos := b.ebuffer.Len()
	checksum := sha1.Sum(plainText)
	b.ebuffer.Write(checksum[:])
	b.ebuffer.Write(plainText)
	out := b.ebuffer.Bytes()
	aesOfb(b.aesBlockCipher, b.iv, out[pos:])
	return out
}

func aesOfb(b cipher.Block, iv []byte, buf []byte) {
	blockSize := 16
	cipher := iv
	for i := 0; i < len(buf); i += blockSize {
		b.Encrypt(cipher[:], cipher[:])
		for j := 0; j < blockSize && i+j < len(buf); j += 1 {
			buf[i+j] = buf[i+j] ^ cipher[j]
		}
	}
}
