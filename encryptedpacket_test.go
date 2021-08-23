package cdclient

import (
	"crypto/aes"
	"encoding/hex"
	"math"
	"testing"
	"time"
)

func BenchmarkFormatEncryptedUdpPacket(bench *testing.B) {
	b, _ := NewEncryptedPacket("username", "password")
	bench.ReportAllocs()
	v := ValueList{
		Metric: &Metric{
			Host:       "example.com",
			Plugin:     "golang",
			Type:       "foobar",
			ValueTypes: []ValueType{DERIVE, GAUGE},
			Interval:   10 * time.Second,
		},
		Time:   time.Unix(1426076671, 123000000), // Wed Mar 11 13:24:31 CET 2015
		Values: []float64{1, math.NaN()},
	}
	err := b.AddValueList(v)
	if err != nil {
		bench.Fatal(err)
	}
	b.Finalize()
	b.Reset()
	bench.ResetTimer()
	for n := 0; n < bench.N; n++ {
		b.AddValueList(v)
		b.Finalize()
		b.Reset()
	}
}

func TestAesOfb(t *testing.T) {
	key := make([]byte, 16)
	iv := make([]byte, 16)
	buf := make([]byte, 27)
	aesBlockCipher, err := aes.NewCipher(key)
	if err != nil {
		t.Fatal(err)
	}

	// We have a no allocation version of this:
	/*
		aesOfb := func (b cipher.Block, iv []byte, buf[]byte) {
			out := &bytes.Buffer{}
			w := &cipher.StreamWriter{S: cipher.NewOFB(b, iv), W: out}
			w.Write(buf)
			copy(buf, out.Bytes())
		}
	*/

	expected := "66e94bd4ef8a2c3b884cfa59ca342b2ef795bd4a52e29ed713d313"

	aesOfb(aesBlockCipher, iv, buf)
	if hex.EncodeToString(buf) != expected {
		t.Fatalf("aesOfb gave an unexpected result")
	}
}
