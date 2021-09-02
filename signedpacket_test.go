package cdclient

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"math"
	"testing"
	"time"
)

func BenchmarkFormatSignedPacket(bench *testing.B) {
	b, _ := NewSignedPacket("username", "password")
	bench.ReportAllocs()
	m := Metric{
		Host:     "example.com",
		Plugin:   "golang",
		Type:     "foobar",
		DSTypes:  []DSType{DERIVE, GAUGE},
		Interval: 10 * time.Second,
	}
	v := ValueList{
		Metric: &m,
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

func TestHmacSha256(t *testing.T) {
	hm1 := hmac.New(sha256.New, []byte("password"))
	hm1.Write([]byte("foobar"))
	h1 := hm1.Sum(nil)

	hm2 := newHmacSha256([]byte("password"))
	hm2.write([]byte("foobar"))
	h2 := hm2.sum()

	hm2.reset()
	hm2.write([]byte("foobar"))
	h3 := hm2.sum()

	h1s := hex.EncodeToString(h1)
	h2s := hex.EncodeToString(h2)
	h3s := hex.EncodeToString(h3)

	if h1s != h2s {
		t.Fatalf("%s != %s", h1s, h2s)
	}
	if h1s != h3s {
		t.Fatalf("%s != %s", h1s, h2s)
	}
}
