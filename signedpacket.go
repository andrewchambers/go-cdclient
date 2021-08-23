package cdclient

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"hash"
)

type SignedPacket struct {
	PlainTextPacket
	username, password []byte
	hmac               *hmacSha256
	signed             bytes.Buffer
}

func NewSignedPacket(username, password string) (*SignedPacket, error) {
	return NewSignedPacketSize(username, password, DefaultBufferSize)
}

func NewSignedPacketSize(username, password string, size int) (*SignedPacket, error) {
	if len(username) > 64 {
		return nil, errors.New("username must be 0-64 characters")
	}
	if len(password) > 64 {
		return nil, errors.New("password must be 0-64 characters")
	}
	b := &SignedPacket{}
	b.hmac = newHmacSha256([]byte(password))
	b.username = []byte(username)
	b.PlainTextPacket.init(size - 36 - len(username))
	return b, nil
}

func (b *SignedPacket) Finalize() []byte {
	if b.buffer.Len() == 0 {
		return nil
	}
	b.signed.Reset()
	b.hmac.reset()
	size := uint16(36 + len(b.username))
	b.hmac.write(b.username)
	b.hmac.write(b.buffer.Bytes())
	tmp := [4]byte{}
	binary.BigEndian.PutUint16(tmp[0:2], uint16(typeSignSHA256))
	binary.BigEndian.PutUint16(tmp[2:4], size)
	b.signed.Write(tmp[:])
	mac := b.hmac.sum()
	b.signed.Write(mac[:])
	b.signed.Write(b.username)
	b.signed.Write(b.buffer.Bytes())
	return b.signed.Bytes()
}

type hmacSha256 struct {
	opad, ipad   [64]byte
	outer, inner hash.Hash
	s1           []byte
	s2           []byte
}

func newHmacSha256(key []byte) *hmacSha256 {
	hm := &hmacSha256{
		outer: sha256.New(),
		inner: sha256.New(),
		s1:    make([]byte, 0, 32),
		s2:    make([]byte, 0, 32),
	}
	blocksize := 64
	if len(key) > blocksize {
		panic("unreachable")
	}
	copy(hm.ipad[:], key)
	copy(hm.opad[:], key)
	for i := 0; i < 64; i++ {
		hm.ipad[i] ^= 0x36
		hm.opad[i] ^= 0x5c
	}
	hm.reset()
	return hm
}

func (hm *hmacSha256) reset() {
	hm.outer.Reset()
	hm.inner.Reset()
	hm.inner.Write(hm.ipad[:])
}

func (h *hmacSha256) sum() []byte {
	h.outer.Write(h.opad[:])
	h.outer.Write(h.inner.Sum(h.s1[:0]))
	return h.outer.Sum(h.s2[:0])
}

func (h *hmacSha256) write(p []byte) (n int, err error) {
	return h.inner.Write(p)
}
